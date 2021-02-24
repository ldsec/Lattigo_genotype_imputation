package lib

import (
	"github.com/ldsec/lattigo/v2/ckks"
)

// DimPatients is the number of patients per ciphertext
var DimPatients = 1004

// NbrTagSnps is the number of target SNP
var NbrTagSnps = 16184

// MatrixNames references each matrix for each SNP
var MatrixNames = []string{"0", "1", "2"}

// LogN is the log2 of the CKKS ring dimension
var LogN uint64 = 10

// Moduli is the ciphertext modulus (29 bits)
var Moduli ckks.Moduli = ckks.Moduli{Qi: []uint64{0x20002801}, Pi: []uint64{}}

// PlaintextModelScale is the value by which the plaintext model(s) coefficients are scaled by
var PlaintextModelScale float64 = 1 << 7

// CiphertextScale is the value by which the encrypted values are scaled by
var CiphertextScale float64 = 1 << 16

// Sigma is the standard deviation of the Gaussian distribution used during the encryption
var Sigma = ckks.DefaultSigma
