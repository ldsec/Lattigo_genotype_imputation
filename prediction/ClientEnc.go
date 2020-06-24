package main

import (
	"encoding/binary"
	"github.com/ldsec/idash19_Task2/prediction/client"
	"github.com/ldsec/idash19_Task2/prediction/lib"
	"log"
	"os"
	"strconv"
)

func main() {
	// **************************** fixed numbers ****************************

	// total number of tag SNPs
	nbrTagSnps := 16184 //TODO can be flexible later

	// **************************** parse arguments ****************************
	var err error
	var refDataPath string
	var nbrGoRoutines, nbrTagSnpsInBatch int

	args := os.Args[1:]

	if len(args) == 0 {
		panic("Need REFDATAPATH, GOROUTINES, BATCHSIZE parameters")
	}

	// path to test ref data (tag SNPs), assigned by client
	refDataPath = string(args[0])

	// number of go routines
	if nbrGoRoutines, err = strconv.Atoi(args[1]); err != nil {
		panic(err)
	}

	// number of tag SNPs per batch in client encryption
	if nbrTagSnpsInBatch, err = strconv.Atoi(args[2]); err != nil {
		panic(err)
	}

	// calculate patient numbers from client files
	DimPatients := lib.GetPatientsNumber(refDataPath)

	log.Printf("[Client] patients number: %v\n", DimPatients)
	log.Println("[Client]: Encryption with: nbrGoRoutines:", nbrGoRoutines, " and batchSize:", nbrTagSnpsInBatch)

	// **************************** initiate client ****************************

	// InitiateClient reads sk and gets the number of patients
	client := client.Client{}
	client.InitiateClient(refDataPath, nbrGoRoutines)

	// **************************** client encryption ****************************

	// Processes and encrypts the client data. Encryption is done in a compressed format :
	// Each ciphertext is of the form (-a * sk + m + e, a)
	// The polynomial 'a' being public, can be generated deterministically from a seeded PRNG.
	// This relaxes the memory footprint of the client encrypted data and the data to be sent by half as
	// only (-a * sk + m + e) parts of the ciphertext need to be stored and sent to the server (along long with the seed).
	// The server can then reconstruct 'a' from the seed.
	encryptedPatientsBatches, seeds := client.ClientPreprocessEncryption(refDataPath, lib.FreqDataPath, nbrTagSnps, nbrTagSnpsInBatch)

	// **************************** write encrypted data ***************************

	lib.PrintMemUsage()
	// time1 := time.Now()

	tmp := make([]byte, 8)

	// Creates a client params file for the unmarshaling of the ciphertexts by the servers
	var fw *os.File
	if fw, err = os.Create(lib.ClientParamsPath); err != nil {
		panic(err)
	}

	// Size of each ciphertext
	dataLen := lib.GetCiphertextDataLenSeeded(encryptedPatientsBatches[0][0], true)
	binary.LittleEndian.PutUint64(tmp, dataLen)
	fw.Write(tmp)

	// Number of Encryptors used by the client
	binary.LittleEndian.PutUint64(tmp, uint64(nbrGoRoutines))
	fw.Write(tmp)

	// Number of SNPTags in each batch
	binary.LittleEndian.PutUint64(tmp, uint64(nbrTagSnpsInBatch))
	fw.Write(tmp)

	// Seeds used by the encryptors to sample the uniform polynomials
	// Will be used by the server to reconstruct the second part of the ciphertexts
	for i := range seeds {
		fw.Write(seeds[i])
	}

	fw.Close()

	// Creates the files containing the compressed ciphertexts
	//numberOfBatches, _ := lib.NbrBatchAndLastBatchSize(nbrTagSnps, nbrTagSnpsInBatch)
	//log.Println("nbrBatches:", numberOfBatches)

	if fw, err = os.Create(lib.ClientEncDataPath); err != nil {
		panic(err)
	}
	defer fw.Close()

	b := make([]byte, dataLen)
	for p := range encryptedPatientsBatches {

		for j := range encryptedPatientsBatches[p] {

			if err = lib.MarshalBinaryCiphertextSeeded32(encryptedPatientsBatches[p][j], b); err != nil {
				panic(err)
			}

			fw.Write(b)
		}
	}

	//time2 := time.Now()
	//fmt.Printf("[Client] writing encrypted data done in %f s\n", time2.Sub(time1).Seconds())
	//lib.PrintMemUsage()
}
