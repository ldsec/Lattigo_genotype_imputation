package client

import (
	"fmt"
	"github.com/ldsec/Lattigo_genotype_imputation/prediction/lib"
	"github.com/ldsec/lattigo/v2/ckks"
	"math"
	"os"
	"sync"
	"time"
)

// Client is a struct storing the scheme parameters keys and other parameters
// for the encryption and decryption.
type Client struct {
	// Scheme Parameters
	params *ckks.Parameters

	// Degree of parallelization
	nbrGoRoutines int

	// Keys
	sk *ckks.SecretKey
}

// InitiateClient populates the Client struct with the scheme parameters, keys and other parameters
// for the encryption and decryption.
func (c *Client) InitiateClient(refDataPath string, nbrGoRoutines int) {

	var err error

	c.nbrGoRoutines = nbrGoRoutines

	if c.params, err = ckks.NewParametersFromModuli(lib.LogN, &lib.Moduli); err != nil {
		panic(err)
	}

	c.params.SetScale(lib.CiphertextScale)

	// read sk
	frSk, _ := os.Open("Keys/SecretKey.binary")
	defer frSk.Close()

	frSkInfo, err := frSk.Stat()
	bufReadSk := make([]byte, frSkInfo.Size())

	if _, err = frSk.Read(bufReadSk); err != nil {
		panic(err)
	}

	c.sk = new(ckks.SecretKey)

	if err = c.sk.UnmarshalBinary(bufReadSk); err != nil {
		panic(err)
	}
}

// ClientPreprocessEncryption processes the data and encrypts it into the correct format.
func (c *Client) ClientPreprocessEncryption(refDataPath string, nbrTagSnps, nbrTagSnpsInBatch int) (encryptedPatientsBatches [][]*ckks.Ciphertext, seeds [][]byte) {

	// batch parameters :
	numberOfBatches, lastBatchSize := lib.NbrBatchAndLastBatchSize(nbrTagSnps, nbrTagSnpsInBatch)

	// Preprocessing ********************************************************************************

	// matrixP: 16184 rows (tag SNPs), 1004 columns (patients)
	matrixP := ReadPatientMatrix(refDataPath, lib.DimPatients, c.nbrGoRoutines)

	// prepare arrays for encryption
	// pvalue[p] is a matrix for batch p, each row is target k in batch p for all patients
	pvalue := make([][][]float64, numberOfBatches)

	// Instantiates the encryptors that will be used by the Go routines
	// Also extracts the seeds used for the uniform polynomials PRNG
	encryptors := make([]*Encryptor, c.nbrGoRoutines)
	seeds = make([][]byte, c.nbrGoRoutines)
	for i := range encryptors {
		encryptors[i] = c.NewEncryptor()
		seeds[i] = make([]byte, 64)
		copy(seeds[i], encryptors[i].seedUniformSampler)
	}

	// Puts the data into an appropriate format for the encryption
	var wg sync.WaitGroup
	wg.Add(c.nbrGoRoutines)
	var start, finish int
	batchSize := numberOfBatches / c.nbrGoRoutines
	for g := 0; g < c.nbrGoRoutines; g++ {
		start = g * batchSize
		finish = (g + 1) * batchSize

		if g == c.nbrGoRoutines-1 {
			finish = numberOfBatches
		}

		go func(start, finish int, encryptor *Encryptor) {

			for p := start; p < finish; p++ {

				nbr := nbrTagSnpsInBatch

				if p == numberOfBatches-1 {
					nbr = lastBatchSize
				}

				// pvalue[p] is a matrix for batch p, each row is target k in batch p for all patients
				pvalue[p] = encryptor.PreprocessData(matrixP, nbr, lib.DimPatients, p)
			}

			wg.Done()
		}(start, finish, encryptors[g])

	}
	wg.Wait()

	// Encryption ********************************************************************************

	// encryptedPatientsBatches[p][k] is a ciphertext, where p is batch number, k is target number in batch p
	encryptedPatientsBatches = make([][]*ckks.Ciphertext, numberOfBatches)

	time3 := time.Now()
	wg.Add(c.nbrGoRoutines)
	for g := 0; g < c.nbrGoRoutines; g++ {

		start = g * batchSize
		finish = (g + 1) * batchSize

		if g == c.nbrGoRoutines-1 {
			finish = numberOfBatches
		}

		go func(start, finish int, encryptor *Encryptor) {

			for p := start; p < finish; p++ {

				nbr := nbrTagSnpsInBatch

				if p == numberOfBatches-1 {
					nbr = lastBatchSize
				}

				// pvalue[p] is a matrix for batch p, each row is target k in batch p for all patients
				// encryptedPatientsBatches[p] is a group of ciphertexts, for all targets in this batch
				encryptedPatientsBatches[p] = encryptor.Encrypt(pvalue[p], nbr)
			}

			wg.Done()
		}(start, finish, encryptors[g])
	}
	wg.Wait()

	time4 := time.Now()
	fmt.Printf("[Client] encryption done in %f s\n", time4.Sub(time3).Seconds())

	return encryptedPatientsBatches, seeds
}

// ClientDecryption reads, decrypts and processes the data from the server.
// predictions[p][i][k]: batch p, coefficient Matrix i, target position k in batch p
func (c *Client) ClientDecryption(predictions [][]*ckks.Ciphertext, nbrTargetSnps, nbrCiphertextInBatch int) (predResAllPatients [][][]float64) {

	nbrCiphertexts := len(predictions[0])

	time1 := time.Now()

	// predicDec[i][k][x]: i-th proba, target k, patient x
	predicDec := make([][][]float64, 3)
	for i := 0; i < 3; i++ {
		predicDec[i] = make([][]float64, nbrTargetSnps)
	}

	// Instantiates the decryptor that will be used by the Go routines
	decryptors := make([]*Decryptor, c.nbrGoRoutines)
	for i := 0; i < c.nbrGoRoutines; i++ {
		decryptors[i] = c.NewDecryptor()
	}

	var wg sync.WaitGroup
	workPerGoRoutine := int(math.Ceil(float64(nbrCiphertexts) / float64(c.nbrGoRoutines)))
	for i := 0; i < 3; i++ {

		wg.Add(c.nbrGoRoutines)
		for g := 0; g < c.nbrGoRoutines; g++ {

			start := g * workPerGoRoutine
			finish := lib.Min((g+1)*workPerGoRoutine, nbrCiphertexts)

			go func(i, start, finish int, decryptor *Decryptor) {

				// target k and k+1 are stored in one complex number as [k + i*(k+1)]
				for j := start; j < finish; j++ {
					predicDec[i][j] = decryptor.Decrypt(lib.DimPatients, predictions[i][j])
				}

				wg.Done()
			}(i, start, finish, decryptors[g])
		}

		wg.Wait()
	}

	//log.Println("Check predicDec:", len(predicDec[0][0])) // should be 1004

	time2 := time.Now()
	fmt.Printf("[Client] decryption done in %f s\n", time2.Sub(time1).Seconds())
	lib.PrintMemUsage()

	// Nomalization

	// predResAllPatients[i][k][m]: probability that predictedSnp=i (from 0 to 2), target=k (from 0 to 80882),patient=m(from  0 to 1004)
	predResAllPatients = make([][][]float64, 3)
	for i := range predResAllPatients {
		predResAllPatients[i] = make([][]float64, nbrTargetSnps)
	}

	for k := 0; k < nbrTargetSnps; k++ {

		// average each

		// three vectors added together
		tmpVector := componentWiseAdd(predicDec[0][k], predicDec[1][k], predicDec[2][k])

		// LHS: target = k-th target in batch p, predicted probability for 0/1/2. LHS is a vector, 1004 length, for all patients
		// RHS: predicDec[p][i][k][x]: batch p, i-th proba, target k in batch p, patient x
		predResAllPatients[0][k] = componentWiseDivide(predicDec[0][k], tmpVector)
		predResAllPatients[1][k] = componentWiseDivide(predicDec[1][k], tmpVector)
		predResAllPatients[2][k] = componentWiseDivide(predicDec[2][k], tmpVector)
	}

	return
}

func componentWiseAdd(a1 []float64, a2 []float64, a3 []float64) []float64 {
	if len(a1) != len(a2) || len(a1) != len(a3) || len(a2) != len(a3) {
		panic("componentWiseAdd failed: vector size not matched")
	}

	res := make([]float64, len(a1))
	for idx := range a1 {
		res[idx] = a1[idx] + a2[idx] + a3[idx]
	}

	return res
}

func componentWiseDivide(dividend []float64, divisor []float64) []float64 {

	dividendSize := len(dividend)
	divisorSize := len(divisor)

	if dividendSize != divisorSize {
		panic("componentWiseDivide failed: vector size not matched")
	}

	res := make([]float64, divisorSize)
	for idx := 0; idx < divisorSize; idx++ {
		res[idx] = dividend[idx] / divisor[idx]
	}

	return res
}
