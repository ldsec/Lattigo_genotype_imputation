package main

import (
	"fmt"
	"github.com/ldsec/Lattigo_genotype_imputation/prediction/lib"
	"github.com/ldsec/lattigo/v2/ckks"
	"os"
	"time"
)

func main() {
	var err error

	time1 := time.Now()

	var params *ckks.Parameters
	if params, err = ckks.NewParametersFromModuli(lib.LogN, &lib.Moduli); err != nil {
		panic(err)
	}

	kgen := ckks.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyGaussian()

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
}
