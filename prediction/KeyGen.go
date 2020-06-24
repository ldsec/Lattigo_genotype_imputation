package main

import (
	"fmt"
	"github.com/ldsec/idash19_Task2/prediction/lib"
	"github.com/ldsec/lattigo/ckks"
	"os"
	"time"
)

func main() {
	var err error

	time1 := time.Now()
	params := lib.Params.Params

	params.Gen()

	kgen := ckks.NewKeyGenerator(&params)
	sk := kgen.GenSecretKeyGaussian(params.Sigma)

	lib.PrintMemUsage()
	time2 := time.Now()
	fmt.Printf("[Key Generation] key generation done %f s\n", time2.Sub(time1).Seconds())

	// Marshal SecretKey
	var fwSk *os.File
	var b []byte

	if fwSk, err = os.Create(lib.KeysPath); err != nil {
		panic(err)
	}
	defer fwSk.Close()

	if b, err = sk.MarshalBinary(); err != nil {
		panic(err)
	}

	fwSk.Write(b)

	//time3 := time.Now()
	//fmt.Printf("[Key Generation] key generation and saving done %f s\n", time3.Sub(time1).Seconds())
}
