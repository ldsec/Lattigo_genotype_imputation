package main

import (
	"bytes"
	"encoding/binary"
	"github.com/ldsec/idash19_Task2/prediction/lib"
	"github.com/ldsec/idash19_Task2/prediction/server"
	"github.com/ldsec/lattigo/ckks"
	"github.com/ldsec/lattigo/ring"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
)

func main() {

	var err error

	// ****************** fixed numbers ****************

	nbrTagSnps := 16184

	// **************************** parse arguments ****************************

	var windowSize, nbrGoRoutines, nbrTargetSnpsInBatch, nbrTargetSnps, DimPatients int

	args := os.Args[1:]
	if len(args) == 0 {
		panic("Need WINDOW, GOROUTINES, BATCHSIZE parameter")
	}

	if windowSize, err = strconv.Atoi(args[0]); err != nil {
		panic(err)
	}

	if nbrGoRoutines, err = strconv.Atoi(args[1]); err != nil {
		panic(err)
	}

	// number of target SNPs per batch in server prediction
	if nbrTargetSnpsInBatch, err = strconv.Atoi(args[2]); err != nil {
		panic(err)
	}

	if nbrTargetSnps, err = strconv.Atoi(args[3]); err != nil {
		panic(err)
	}

    if DimPatients, err = strconv.Atoi(args[4]); err != nil {
       panic(err) 
    }

	log.Println("[Server]: Prediction for nbrTags:", nbrTargetSnps, " with: nbrGoRoutines:", nbrGoRoutines, " and nbrTargetSnpsInBatch:", nbrTargetSnpsInBatch, " and patients:", DimPatients)


	params := lib.Params.Params

	params.Gen()

	// Retrieves the client encryption params
	// : dataLen of each ciphertext
	// : nbrEncryptors used
	// : nbrTatSNP in each batch (which allows to compute the number of batches)
	// : seeds used by the encryptors
	var fr *os.File
	if fr, err = os.Open(lib.ClientParamsPath); err != nil {
		panic(err)
	}
	defer fr.Close()

	frInfo, err := fr.Stat()
	frSize := frInfo.Size()
	bufRead := make([]byte, frSize)
	if _, err = fr.Read(bufRead); err != nil {
		panic(err)
	}

	datalen := int(binary.LittleEndian.Uint64(bufRead[:8]))
	nbrEncryptorsClient := int(binary.LittleEndian.Uint64(bufRead[8:16]))
	nbrTagSnpsInBatch := int(binary.LittleEndian.Uint64(bufRead[16:24]))

	seeds := make([][]byte, (frSize-16)/64)
	for i := range seeds {
		seeds[i] = make([]byte, 64)
		copy(seeds[i], bufRead[24+i*64:24+(i+1)*64])
	}

	// contruct slot for reading client encrypted data
	encryptedPatients := make([]*ckks.Ciphertext, nbrTagSnps) // nbrTagSnps ciphertexts

	//time1 := time.Now()

	if fr, err = os.Open(lib.ClientEncDataPath); err != nil {
		panic(err)
	}
	defer fr.Close()

	// Unmarchals the part -a * sk + m + e of the ciphertext
	for tag := 0; tag < nbrTagSnps; tag++ {
		cipherPool := make([]byte, datalen)
		fr.Read(cipherPool)

		encryptedPatients[tag] = new(ckks.Ciphertext)
		if err = lib.UnmarshalBinaryCiphertextSeeded32(encryptedPatients[tag], cipherPool); err != nil {
			log.Println("err position:", tag)
			panic(err)
		}
	}

	//time2 := time.Now()
	//fmt.Printf("[Server] unmarshaling ciphertext data done in %f s\n", time2.Sub(time1).Seconds())
	//lib.PrintMemUsage()

	//time1 = time.Now()

	// Reconstruct the 'a' second part of the ciphertext
	var ringContext *ring.Context
	if ringContext, err = ring.NewContextWithParams(params.N, params.Qi); err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(nbrEncryptorsClient)
	var start, finish int
	numberOfBatches, _ := lib.NbrBatchAndLastBatchSize(nbrTagSnps, nbrTagSnpsInBatch)
	batchSize := numberOfBatches / nbrEncryptorsClient
	for g := 0; g < nbrEncryptorsClient; g++ {
		start = g * batchSize
		finish = (g + 1) * batchSize

		if g == nbrEncryptorsClient-1 {
			finish = numberOfBatches
		}

		go func(start, finish int, seed []byte) {

			crpGen := ring.NewCRPGenerator(nil, ringContext)
			crpGen.Seed(seed)

			for p := start; p < finish; p++ {

				end := (p + 1) * nbrTagSnpsInBatch
				if p == numberOfBatches-1 {
					end = nbrTagSnps
				}

				for i := p * nbrTagSnpsInBatch; i < end; i++ {
					encryptedPatients[i].Value()[1] = crpGen.ClockUniformNew()

				}
			}
			wg.Done()
		}(start, finish, seeds[g])

	}
	wg.Wait()

	//time2 = time.Now()
	//fmt.Printf("[Server] ciphertext reconstruction from seed done in %f s\n", time2.Sub(time1).Seconds())
	//lib.PrintMemUsage()

	// read mapping table
	MappingList := server.ReadMappingTable(lib.ServerMappingTablePath, nbrGoRoutines)

	// predictions [][][]*ckks.Ciphertext: predictions[p][i][k] batch p, matrixRi, target k (one ciphertext, for all patients)
	predictionsSave := server.Prediction(DimPatients, windowSize, lib.MatrixRPath(args[0]), MappingList, encryptedPatients, nbrGoRoutines, nbrTargetSnps, nbrTargetSnpsInBatch)
	lib.PrintMemUsage()

	//time3 := time.Now()

	// We need 3 ciphertext to store each SNPtarget, and we pack twice as many values using the imaginary part
	nbrCiphertexts := nbrTargetSnps
	dataLen := lib.GetCiphertextDataLen(predictionsSave[0][0], true)

	nbrCiphertextsInFile := int(float64(lib.MaxResultFileSizeMB) / (float64(dataLen) / (1000000)))
	nbrFiles := int(math.Ceil(float64(nbrCiphertexts) / float64(nbrCiphertextsInFile)))

	var fpRes *os.File
	if fpRes, err = os.Create(lib.ServerEncParameters); err != nil {
		panic(err)
	}

	bufRes := new(bytes.Buffer)

	binary.Write(bufRes, binary.LittleEndian, uint64(nbrTargetSnps))
	binary.Write(bufRes, binary.LittleEndian, uint64(nbrCiphertexts))
	binary.Write(bufRes, binary.LittleEndian, uint64(nbrFiles))

	fpRes.Write(bufRes.Bytes())
	fpRes.Close()

	b := make([]byte, dataLen)

	// For each SNP tag
	for i := 0; i < 3; i++ {

		// We store the result in files of maximum lib.MaxResultFileSizeMB MB
		for number := 0; number < nbrFiles; number++ {

			var fpRes *os.File
			if fpRes, err = os.Create(lib.ResDataPath(i, number)); err != nil {
				panic(err)
			}

			bufRes := new(bytes.Buffer)

			start := number * nbrCiphertextsInFile
			finish := lib.Min((number+1)*nbrCiphertextsInFile, nbrCiphertexts)

			binary.Write(bufRes, binary.LittleEndian, uint64(finish-start))
			binary.Write(bufRes, binary.LittleEndian, dataLen)
			fpRes.Write(bufRes.Bytes())

			for idx := start; idx < finish; idx++ {

				if err = lib.MarshalBinaryCiphertext32(predictionsSave[i][idx], b); err != nil {
					panic(err)
				}

				fpRes.Write(b)
			}

			fpRes.Close()
		}
	}

	//time4 := time.Now()

	//log.Printf("[Server] writing encrypted result done in %f s\n", time4.Sub(time3).Seconds())
	//lib.PrintMemUsage()
}
