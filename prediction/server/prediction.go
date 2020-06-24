package server

import (
	"github.com/ldsec/idash19_Task2/prediction/lib"
	"github.com/ldsec/lattigo/ckks"
	"github.com/ldsec/lattigo/ring"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ReadMappingTable -
func ReadMappingTable(filename string, nbrGoRoutines int) (MappingList []int64) {

	// read in table
	metaMatrixTable := lib.FileToString("prediction_data/mapping_table.txt")

	MappingList = make([]int64, len(metaMatrixTable))

	workPerGoRoutine := len(metaMatrixTable) / nbrGoRoutines

	var wg sync.WaitGroup
	wg.Add(nbrGoRoutines)
	for idx := 0; idx < nbrGoRoutines; idx++ {

		start := idx * workPerGoRoutine
		finish := (idx + 1) * workPerGoRoutine

		if idx == nbrGoRoutines-1 {
			finish = len(metaMatrixTable)
		}

		go func(start, finish int) {

			for i := start; i < finish; i++ {

				strarray := strings.Fields(strings.TrimSpace(metaMatrixTable[i]))

				if tmp, err := strconv.ParseInt(strarray[0], 10, 64); err == nil {
					MappingList[i] = tmp
				}
			}
			wg.Done()
		}(start, finish)

	}
	wg.Wait()

	return
}

// ReadCoefficients reads the model
func ReadCoefficients(windowSize int, matrixRPath string, nbrLabels, nbrPositions, nbrGoRoutines int) (MatrixRs [][][]float64) {

	// no need to pre-multiply now
	// in fastsquare version, multiply beta on plaintext values
	// otherwise do nothing

	MatrixRs = make([][][]float64, nbrLabels)
	for i := range MatrixRs {
		MatrixRs[i] = make([][]float64, windowSize)
	}

	workPerGoRoutine := nbrLabels / nbrGoRoutines

	var wg sync.WaitGroup
	wg.Add(nbrGoRoutines)
	for i := 0; i < nbrGoRoutines; i++ {

		start := i * workPerGoRoutine
		finish := (i + 1) * workPerGoRoutine

		if i == nbrGoRoutines-1 {
			finish = nbrLabels
		}

		go func(start, finish int) {

			for i := start; i < finish; i++ {
				/* Read in coef matrix R, saved in ArrayR */
				MatrixR := lib.FileToString(matrixRPath + strconv.Itoa(i) + ".csv")

				//ArrayR := [1<<5][]float64{}
				ArrayR := make([][]float64, windowSize)
				var cnt uint64
				for idx := range MatrixR {

					strarray := strings.Fields(strings.TrimSpace(MatrixR[idx]))

					ArrayR[cnt] = make([]float64, nbrPositions)

					for i := range strarray {

						if tmp, err := strconv.ParseFloat(strarray[i], 64); err == nil {
							if i >= nbrPositions {
								continue
							} else {
								ArrayR[cnt][i] = tmp
							}
						}
					}
					cnt++
				}

				MatrixRs[i] = ArrayR
			}

			wg.Done()

		}(start, finish)
	}
	wg.Wait()

	return
}

// Predictor is a struct storing the data and object necessary to evaluate the model on the encrypted data.
type Predictor struct {
	modelParams *lib.ModelParams
	evaluator   ckks.Evaluator
	context     *ring.Context
	allOnes     *ckks.Ciphertext
}

// NewPredictor creates a new rpedictor.
func NewPredictor(modelParams *lib.ModelParams) (p *Predictor) {
	p = new(Predictor)
	p.modelParams = modelParams.Copy()
	params := &p.modelParams.Params
	p.evaluator = ckks.NewEvaluator(params)
	p.context, _ = ring.NewContextWithParams(1<<params.LogN, params.Qi)

	// The goal of this step is to create a ciphertext encrypted with the zerokey (i.e. a plaintext but of the type "ciphertext"
	// that encodes a vector of all ones in the NTT domain (since all ciphertexts are by default in the NTT domain).
	p.allOnes = ckks.NewCiphertext(&modelParams.Params, 1, 0, p.modelParams.PlaintextModelScale*p.modelParams.Params.Scale)

	for i, qi := range p.context.Modulus {

		p1 := p.allOnes.Value()[0].Coeffs[i]

		value := scaleUpExact(1, p.modelParams.Params.Scale, qi)

		for j := uint64(0); j < p.context.N; j++ {
			p1[j] = value
		}
	}

	p.context.NTT(p.allOnes.Value()[0], p.allOnes.Value()[0])

	return
}

// Predict evaluates the model of the server on the encrypted data.
func (p *Predictor) Predict(arrayR [][]float64, MappingList []int64, encryptedPatients []*ckks.Ciphertext, nbrCoeffs, nbrTargetSnpsInBatch, batchSize, batchIndex int) (predictionsInBatch []*ckks.Ciphertext, durationPred time.Duration) {

	startTime := time.Now()

	ptScale := p.modelParams.PlaintextModelScale

    predictionsInBatch = make([]*ckks.Ciphertext, batchSize)

    for target := 0; target < batchSize; target++ {
		predictionsInBatch[target] = ckks.NewCiphertext(&p.modelParams.Params, 1, 0, p.modelParams.PlaintextModelScale*p.modelParams.Params.Scale)
	}
	coeffsAllOne := p.allOnes.Value()[0].Coeffs[0]

    // prediction for every target position

	// TODO can parallel on targets
	var encryptedTagSnps []*ckks.Ciphertext
	for target := 0; target < batchSize; target++ {

		// select 32 - 1 ciphertexts from encryptedPatients
		// pass mapping table for these batchSize targets
		// st := findStartTag(target + nbrTargetSnpsInBatch * batchIndex)  // the biggest reference ID that is smaller than target ID

		st := int(MappingList[target])

		bound := len(encryptedPatients) // nbrTagSnps in total
		if (st == -1) || (st < int(nbrCoeffs/2)-1) {
			// most left one or left window overflow
			encryptedTagSnps = encryptedPatients[0 : nbrCoeffs-1]

		} else if (st == -2) || (st > bound-nbrCoeffs/2) {
			// most right one or right window overflow
			encryptedTagSnps = encryptedPatients[bound-nbrCoeffs+1 : bound]

		} else {
			// normal case
			encryptedTagSnps = encryptedPatients[st-(nbrCoeffs/2)+1 : st+(nbrCoeffs/2)]
		}

		// for a given target
		// encoding plaintext (32 plaintext, each plaintext [coef] is replicated with the coef-th value of target)
		// multiplication with plaintext (32 mult)
		// additions (add 32 ciphertexts together)

		var ct *ckks.Ciphertext
		var weight float64
		for coef := 0; coef < nbrCoeffs; coef++ {

			// We extract the corresponding coefficient of the model
			weight = arrayR[coef][target+nbrTargetSnpsInBatch*batchIndex]

			if coef == 0 {

				// The first step is the multiplication of a ciphertext encrypting all one with the weight.
				// This is done by creating a new empty ciphertext set to zero and adding the scaled weight on it.
				
				for i, qi := range p.context.Modulus {

					bredParams := p.context.GetBredParams()[i]

					p1 := predictionsInBatch[target].Value()[0].Coeffs[i]

					// Scales the weight and puts it in the Montgomery domain
					// r * Delta * weight
					value := ring.MForm(scaleUpExact(weight, ptScale, qi), qi, bredParams)

					for j := uint64(0); j < p.context.N; j++ {
                        p1[j] = coeffsAllOne[j] * value
					}

				}

			} else {

				ct = encryptedTagSnps[coef-1]

				// We multiply the ciphertext by the weight and adds the result on the accumulator without modular reduction
				for i, qi := range p.context.Modulus {

					for u := 0; u < 2; u++ {

						bredParams := p.context.GetBredParams()[i]

						p0 := ct.Value()[u].Coeffs[i]
						p1 := predictionsInBatch[target].Value()[u].Coeffs[i]

						// Scales the weight and puts it in the Montgomery domain
						// r * Delta * weight
						value := ring.MForm(scaleUpExact(weight, ptScale, qi), qi, bredParams)

						// Montgomery multiplication without modular reduction
						// sum(ai * r * bi) = r * sum(ai * bi)
						for j := uint64(0); j < p.context.N; j++ {
							p1[j] += p0[j] * value
						}
					}
				}
			}
		}

		// Montgomery modular reduction of r * sum(ai * bi)
		p.context.InvMForm(predictionsInBatch[target].Value()[0], predictionsInBatch[target].Value()[0])
		p.context.InvMForm(predictionsInBatch[target].Value()[1], predictionsInBatch[target].Value()[1])

		durationPred = time.Now().Sub(startTime)
	}

	return predictionsInBatch, durationPred
}

func scaleUpExact(value float64, n float64, q uint64) (res uint64) {

	var isNegative bool
	var xFlo *big.Float
	var xInt *big.Int

	isNegative = false
	if value < 0 {
		isNegative = true
		xFlo = big.NewFloat(-n * value)
	} else {
		xFlo = big.NewFloat(n * value)
	}

	xFlo.Add(xFlo, big.NewFloat(0.5))

	xInt = new(big.Int)
	xFlo.Int(xInt)
	xInt.Mod(xInt, ring.NewUint(q))

	res = xInt.Uint64()

	if isNegative {
		res = q - res
	}

	return
}
