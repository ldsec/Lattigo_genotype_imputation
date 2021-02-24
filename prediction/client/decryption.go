package client

import (
	"github.com/ldsec/lattigo/v2/ckks"
	"math"
)

// Decryptor is a struct storing the necessary object to decrypt and decode ciphertexts.
type Decryptor struct {
	decryptor ckks.Decryptor
	encoder   ckks.Encoder
	plaintext *ckks.Plaintext
}

// NewDecryptor creates a new Decryptor.
func (c *Client) NewDecryptor() (decryptor *Decryptor) {
	decryptor = new(Decryptor)
	decryptor.decryptor = ckks.NewDecryptor(c.params, c.sk)
	decryptor.encoder = ckks.NewEncoder(c.params)
	decryptor.plaintext = ckks.NewPlaintext(c.params, 0, 0)
	return
}

// Decrypt decrypts and decodes a ciphertext and applies the necessary post-processing.
func (d *Decryptor) Decrypt(nbrPatients int, ciphertext *ckks.Ciphertext) (pred []float64) {

	// Decryption & decoding process
	d.decryptor.Decrypt(ciphertext, d.plaintext)
	valuesTest := d.encoder.DecodeCoeffs(d.plaintext)

	pred = make([]float64, nbrPatients)
	for j := 0; j < nbrPatients; j++ {
		pred[j] = 1 / (math.Exp(-valuesTest[j]) + 1)
	}

	return
}
