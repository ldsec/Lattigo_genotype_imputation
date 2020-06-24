package main

import (
	"encoding/binary"
	"github.com/ldsec/idash19_Task2/prediction/client"
	"github.com/ldsec/idash19_Task2/prediction/lib"
	"github.com/ldsec/lattigo/ckks"
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
	
    params := lib.Params.Params
	params.Gen()

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
    DimPatients := lib.GetPatientsNumber(refDataPath)
    //log.Println("DimPatients", DimPatients)

	// **************************** read encrypted prediction result ****************************

	// Read predictions (ciphertext) from EncRes.binary
	//time1 := time.Now()

	// *********** recover predictions ***********
	// read predictionsSave[target][i] from ciphertext file: target from 0 to 80882, i=0,1,2

	predictions := make([][]*ckks.Ciphertext, 3)
	for i := 0; i < 3; i++ {
		predictions[i] = make([]*ckks.Ciphertext, nbrCiphertext) // i=0,1,2
	}

	for i := 0; i < 3; i++ {

		var k uint64

		for number := 0; number < nbrFiles; number++ {

			var fr *os.File
			if fr, err = os.Open(lib.ResDataPath(i, number)); err != nil {
				panic(err)
			}

			frInfo, err := fr.Stat()
			bufRead := make([]byte, frInfo.Size())
			if _, err := fr.Read(bufRead); err != nil {
				panic(err)
			}

			var ptr uint64
			nbrCiphertextsInFile := int(binary.LittleEndian.Uint64(bufRead[ptr : ptr+8]))
			dataLen := binary.LittleEndian.Uint64(bufRead[ptr+8 : ptr+16])
			ptr += 16

			for idx := 0; idx < nbrCiphertextsInFile; idx++ {

                predictions[i][k] = ckks.NewCiphertext(&params, 1, 0, 0)
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

	//time2 := time.Now()
	//fmt.Printf("[Client] reading encrypted result done in %f s\n", time2.Sub(time1).Seconds())
	//lib.PrintMemUsage()

	// **************************** client decryption ****************************
	predResAllPatients := client.ClientDecryption(predictions, nbrTargetSnps, nbrCiphertextInBatch) // probably wrong
	// predResAllPatients[i][k][m]:
	// probability that predictedSnp=i (from 0 to 2), target=k (all targets, from 0 to nbrTargetSnps-1),patient=m(from  0 to nbrPatiens-1)

	// **************************** write decrypted result ****************************

	//time2 = time.Now()

	var fp *os.File
	if fp, err = os.Create(lib.ClientResDataPath); err != nil {
		panic(err)
	}

	workPerGoRoutine := int(math.Ceil(float64(DimPatients*nbrTargetSnps) / float64(nbrGoRoutines)))

	b := make([][]byte, nbrGoRoutines)

	var wg sync.WaitGroup
	wg.Add(nbrGoRoutines)
	for g := 0; g < nbrGoRoutines; g++ {

		start := g * workPerGoRoutine
		finish := (g + 1) * workPerGoRoutine

		if g == nbrGoRoutines-1 {
			finish = DimPatients * nbrTargetSnps
		}

		b[g] = make([]byte, 3*(finish-start)*2)

		go func(start, finish int, b []byte) {

			var idx, patient int

			var x, y, z uint16

			for row, i := start, 0; row < finish; row, i = row+1, i+1 {

				idx = row % nbrTargetSnps
				patient = row / nbrTargetSnps

				x = uint16(predResAllPatients[0][idx][patient] * 65536)
				y = uint16(predResAllPatients[1][idx][patient] * 65536)
				z = uint16(predResAllPatients[2][idx][patient] * 65536)

				b[0+6*i] = uint8(x)
				b[1+6*i] = uint8(x >> 8)
				b[2+6*i] = uint8(y)
				b[3+6*i] = uint8(y >> 8)
				b[4+6*i] = uint8(z)
				b[5+6*i] = uint8(z >> 8)
			}
			wg.Done()
		}(start, finish, b[g])
	}

	wg.Wait()

	//log.Println("write bytes:", 3*DimPatients*nbrTargetSnps*2)
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
	// print all target IDs for a patient

	//time3 := time.Now()
	//fmt.Printf("[Client] results written in %f s\n", time3.Sub(time2).Seconds())

	//lib.PrintMemUsage()

	//fmt.Println("[Client] results for all patients stored in results/ypred.binary")
}
