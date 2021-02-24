package client

import (
	"crypto/rand"
	"encoding/csv"
	"github.com/ldsec/lattigo/v2/ckks"
	"github.com/ldsec/lattigo/v2/ring"
	"github.com/ldsec/lattigo/v2/utils"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

var err error

// ReadPatientMatrix reads the patient matrix and processes it.
// TODO: change targetPath to hardcoded data path
// TODO: unify var name format
// In real test, refPath is the path of idash reserved file.
func ReadPatientMatrix(refDataPath string, nbrPatients, nbrGoRoutines int) (metaArrayP [][]float64) {

	var wg sync.WaitGroup

	metaMatrixP := []string{}

	/* NEW VERSION: Read in original patient matrix directly, save in metaArrayP */
	// TODO: not sure if there will be memory problem for (80000+ SNPs, 1000+ patients)
	metadataP, err := os.Open(refDataPath)
	if err != nil {
		log.Println("Data path is wrong: reading failed")
	}
	r := csv.NewReader(metadataP)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		metaMatrixP = append(metaMatrixP, record[0])
	}

	// store pure data of all patients and all observed positions
	// Note: metaArrayP is just reading data part, no transformation
	// metaArrayP: row --- ref positions, col --- patients
	nbrRefPositions := len(metaMatrixP)
	metaArrayP = make([][]float64, nbrRefPositions)
	for i := range metaArrayP {
		metaArrayP[i] = make([]float64, nbrPatients)
	}

	// store all observed positions on chromosome one
	// will be used to find submatrix
	refPositionList := make([]int64, nbrRefPositions)

	workPerGoRoutine := len(metaMatrixP) / nbrGoRoutines

	wg.Add(nbrGoRoutines)
	for idx := 0; idx < nbrGoRoutines; idx++ {

		start := idx * workPerGoRoutine
		finish := (idx + 1) * workPerGoRoutine

		if idx == nbrGoRoutines-1 {
			finish = len(metaMatrixP)
		}

		go func(start, finish int) {

			for idx := start; idx < finish; idx++ {

				strarray := strings.Fields(strings.TrimSpace(metaMatrixP[idx]))

				for i := range strarray {
					if i == 1 { // get position idx
						if tmp, err := strconv.ParseInt(strarray[i], 10, 64); err == nil {
							refPositionList[idx] = tmp
						}
					} else if i < 4 { // jump over chromosome, start position, end position, ID.
						continue
					} else if tmp, err := strconv.ParseFloat(strarray[i], 64); err == nil {
						metaArrayP[idx][i-4] = tmp
					}
				}
			}

			wg.Done()
		}(start, finish)
	}

	wg.Wait()

	return metaArrayP

}

// Encryptor is a struct storing the necessary objects and data to encode the patient data on a plaintext and and encrypt it.
type Encryptor struct {
	params             *ckks.Parameters
	sk                 *ring.Poly
	ringQ              *ring.Ring
	encoder            ckks.Encoder
	tmpPt              *ckks.Plaintext
	crpGen             *ring.UniformSampler
	gauGen             *ring.GaussianSampler
	seedUniformSampler []byte
	polypool           *ring.Poly
}

// NewEncryptor creates a new Encryptor which is thread safe.
func (c *Client) NewEncryptor() (enc *Encryptor) {
	enc = new(Encryptor)

	enc.params = c.params.Copy()

	enc.sk = c.sk.Get().CopyNew()

	if enc.ringQ, err = ring.NewRing(c.params.N(), c.params.Qi()); err != nil {
		panic(err)
	}

	enc.seedUniformSampler = make([]byte, 64)
	if _, err := rand.Read(enc.seedUniformSampler); err != nil {
		panic("crypto rand error")
	}

	prngUniform, err := utils.NewKeyedPRNG(enc.seedUniformSampler)
	if err != nil {
		panic(err)
	}

	bytes := make([]byte, 64)
	if _, err := rand.Read(bytes); err != nil {
		panic("crypto rand error")
	}

	prngGaussian, err := utils.NewKeyedPRNG(bytes)
	if err != nil {
		panic(err)
	}

	enc.crpGen = ring.NewUniformSampler(prngUniform, enc.ringQ)
	enc.gauGen = ring.NewGaussianSampler(prngGaussian, enc.ringQ, ckks.DefaultSigma, 19)

	enc.polypool = enc.ringQ.NewPoly()

	enc.encoder = ckks.NewEncoder(c.params)
	enc.tmpPt = ckks.NewPlaintext(c.params, c.params.MaxLevel(), c.params.Scale())

	return
}

// Encrypt encodes and encrypts a pre-processed matrix of patient data.
func (enc *Encryptor) Encrypt(valueArray [][]float64, nbrTagSnpsInBatch int) (ciphertexts []*ckks.Ciphertext) {

	// Ciphertexts pool
	ciphertexts = make([]*ckks.Ciphertext, nbrTagSnpsInBatch)

	var tmpCt *ckks.Ciphertext

	// Generate patient encrypted data for multiplying Ri
	for tag := 0; tag < nbrTagSnpsInBatch; tag++ {

		// Encodes the vector on the plaintext m
		enc.encoder.EncodeCoeffs(valueArray[tag], enc.tmpPt)

		// Creates a ciphertext of degree 0 (only the first element needs to be stored as the second element is generated from a seed)
		tmpCt = &ckks.Ciphertext{Element: &ckks.Element{}}
		tmpCt.SetScale(enc.params.Scale())
		tmpCt.SetValue(make([]*ring.Poly, 1))
		tmpCt.Value()[0] = enc.ringQ.NewPoly()

		// Encrypts the plaintext on the ciphertext :
		// ct1 = a (generated from the PRNG, only the seed of the PRNG is stored)
		// ct0 = -a * sk + e + m

		// samples NTT(a)
		enc.crpGen.Read(enc.polypool)

		// comptues NTT(-a*sk)
		enc.ringQ.MulCoeffsMontgomeryAndSub(enc.polypool, enc.sk, tmpCt.Value()[0])

		// computes e + m
		// V1 (uses threadsafe PRNG for gaussian sampling):
		enc.gauGen.ReadAndAdd(enc.tmpPt.Value()[0])

		// NTT(e+m)
		enc.ringQ.NTT(enc.tmpPt.Value()[0], enc.tmpPt.Value()[0])

		// computes NTT(-a *sk) + NTT(e + m)
		enc.ringQ.Add(tmpCt.Value()[0], enc.tmpPt.Value()[0], tmpCt.Value()[0])

		ciphertexts[tag] = tmpCt
	}

	return
}

// PreprocessData processes the matrix of patient data and format it into the desired format for the encoding and the encryption.
func (enc *Encryptor) PreprocessData(metaArrayP [][]float64, currentBatchSize, nbrPatients, batchIndex int) (valueArray [][]float64) {

	// Construct nbrTagSnps slots for patient data to be encrypted, each slot is one position for 1004 patients
	valueArray = make([][]float64, currentBatchSize)

	// read each batch in valueArray, each row of valueArray is a ciphertext of one tag position for 1004 patients
	for i := range valueArray {
		valueArray[i] = make([]float64, enc.params.N())
		for j := 0; j < nbrPatients; j++ {
			valueArray[i][j] = 1.0 + metaArrayP[batchIndex*currentBatchSize+i][j] // remember to + 1
		}
	}

	return
}
