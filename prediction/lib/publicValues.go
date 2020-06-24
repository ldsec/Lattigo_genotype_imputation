package lib

import (
	"github.com/ldsec/lattigo/ckks"
)

// var DimPatients = 135 // All 1004; AMR 135; EUR 210; AFR 272

// MatrixNames
var MatrixNames = []string{"0", "1", "2"}

// ModelParams is a struct storing the parameters for the Model.
type ModelParams struct {
	PlaintextModelScale float64         // Encoding scale of the model coefficients
	Params              ckks.Parameters // Scheme parameters
}

// Copy creates a new ModelParams which is a copy of the target ModelParams
func (m *ModelParams) Copy() (copy *ModelParams) {
	copy = new(ModelParams)
	copy.PlaintextModelScale = m.PlaintextModelScale
	copy.Params = *m.Params.Copy()
	return
}

// Gen generates the internal scheme parameters of the target ModelParams
func (m *ModelParams) Gen() {
	m.Params.Gen()
}

// Linear Model
var Params = ModelParams{
	1 << 7,
	ckks.Parameters{
		LogN:     10,
		LogSlots: 9,
		LogModuli: ckks.LogModuli{
			LogQi: []uint64{29},
			LogPi: []uint64{},
		},
		Scale: 1 << 16,
		Sigma: 3.2},
}
