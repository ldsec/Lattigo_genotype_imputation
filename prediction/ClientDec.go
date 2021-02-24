package main

import (
	"encoding/binary"
	"github.com/ldsec/Lattigo_genotype_imputation/prediction/client"
	"github.com/ldsec/Lattigo_genotype_imputation/prediction/lib"
	"github.com/ldsec/lattigo/v2/ckks"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
)

func main() {

	var err error

	// **************************** parse arguments ****************************

	var nbrGoRoutines, nbrCiphertextInBatch int
	var refDataPath string

	args := os.Args[1:]
	if len(args) == 0 {
		panic("Need REFDATAPATH, GOROUTINES, BATCHSIZE parameters")
	}
	refDataPath = string(args[0])

	if nbrGoRoutines, err = strconv.Atoi(args[1]); err != nil {
		panic(err)
	}

	// number of target SNPs per batch in client decryption
	if nbrCiphertextInBatch, err = strconv.Atoi(args[2]); err != nil {
		panic(err)
	} // should be consistent with pred batch size

	// **************************** initiate client ****************************

	params, err := ckks.NewParametersFromModuli(lib.LogN, &lib.Moduli)
	if err != nil {
		panic(err)
	}

	client := client.Client{}
	log.Println("[Client]: Decryption with: nbrGoRoutines:", nbrGoRoutines, " and nbrBatches:", nbrCiphertextInBatch)
	client.InitiateClient(refDataPath, nbrGoRoutines)

	var fr *os.File
	if fr, err = os.Open(lib.ServerEncParameters); err != nil {
		panic(err)
	}

	frInfo, err := fr.Stat()
	bufRead := make([]byte, frInfo.Size())
	if _, err := fr.Read(bufRead); err != nil {
		panic(err)
	}
	nbrTargetSnps := int(binary.LittleEndian.Uint64(bufRead[:8]))
	nbrCiphertext := int(binary.LittleEndian.Uint64(bufRead[8:16]))
	nbrFiles := int(binary.LittleEndian.Uint64(bufRead[16:24]))

	// *********** recover predictions ***********
	// read predictionsSave[target][i] from ciphertext file: target from 0 to 80882, i=0,1,2

	predictions := make([][]*ckks.Ciphertext, 3)
	for i := 0; i < 3; i++ { // i=0,1,2 (3 SNP)
		predictions[i] = make([]*ckks.Ciphertext, nbrCiphertext)
	}

	for i := 0; i < 3; i++ {

		var k uint64

		for number := 0; number < nbrFiles; number++ {

			var fr *os.File
			if fr, err = os.Open(lib.ResDataPath(i, number)); err != nil {
				panic(err)
			}

			frInfo, err := fr.Stat()
			bufRead := make([]byte, frInfo.Size()) // create a buffer of the file-size
			if _, err := fr.Read(bufRead); err != nil {
				panic(err)
			}

			var ptr uint64
			nbrCiphertextsInFile := int(binary.LittleEndian.Uint64(bufRead[ptr : ptr+8])) // number of ciphertexts in the file
			dataLen := binary.LittleEndian.Uint64(bufRead[ptr+8 : ptr+16])                // size of each ciphertext
			ptr += 16

			for idx := 0; idx < nbrCiphertextsInFile; idx++ {

				// Alocates and populates the ciphertexts
				predictions[i][k] = ckks.NewCiphertext(params, 1, 0, 0)
				if err = lib.UnmarshalBinaryCiphertext32(predictions[i][k], bufRead[ptr:ptr+dataLen]); err != nil {
					panic(err)
				}

				ptr += dataLen
				k++
			}

			bufRead = nil

			fr.Close()
		}
	}

	// **************************** client decryption ****************************
	predResAllPatients := client.ClientDecryption(predictions, nbrTargetSnps, nbrCiphertextInBatch) // probably wrong
	// predResAllPatients[i][k][m]:
	// probability that predictedSnp=i (from 0 to 2), target=k (all targets, from 0 to nbrTargetSnps-1),patient=m(from  0 to nbrPatiens-1)

	// **************************** write decrypted result ****************************

	var fp *os.File
	if fp, err = os.Create(lib.ClientResDataPath); err != nil {
		panic(err)
	}

	workPerGoRoutine := int(math.Ceil(float64(lib.DimPatients*nbrTargetSnps) / float64(nbrGoRoutines)))

	b := make([][]byte, nbrGoRoutines)

	var wg sync.WaitGroup
	wg.Add(nbrGoRoutines)
	for g := 0; g < nbrGoRoutines; g++ {

		start := g * workPerGoRoutine
		finish := (g + 1) * workPerGoRoutine

		if g == nbrGoRoutines-1 {
			finish = lib.DimPatients * nbrTargetSnps
		}

		b[g] = make([]byte, 3*(finish-start)*2)

		go func(start, finish int, b []byte) {

			var idx0, idx1, patient int

			var x, y, z uint16

			for row, i := start, 0; row < finish; row, i = row+1, i+1 {

				idx0 = row % nbrTargetSnps
				idx1 = 6 * i
				patient = row / nbrTargetSnps

				// encode float value on 2 uint8
				x = uint16(predResAllPatients[0][idx0][patient] * 65536)
				y = uint16(predResAllPatients[1][idx0][patient] * 65536)
				z = uint16(predResAllPatients[2][idx0][patient] * 65536)

				b[0+idx1] = uint8(x)
				b[1+idx1] = uint8(x >> 8)
				b[2+idx1] = uint8(y)
				b[3+idx1] = uint8(y >> 8)
				b[4+idx1] = uint8(z)
				b[5+idx1] = uint8(z >> 8)
			}
			wg.Done()
		}(start, finish, b[g])
	}

	wg.Wait()

	for i := range b {
		if _, err = fp.Write(b[i]); err != nil {
			panic(err)
		}
	}

	if err = fp.Close(); err != nil {
		panic(err)
	}

	// result in plaintext written in the form:
	// 1st column - patient number; 2nd column - target ID; 3rd, 4th, 5th: column - probabilities
}
