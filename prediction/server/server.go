package server

import (
	"github.com/ldsec/idash19_Task2/prediction/lib"
	"github.com/ldsec/lattigo/ckks"
	"log"
	"math"
	"sync"
	"time"
)

// Prediction evaluates the model on the client data.
func Prediction(nbrPatients, windowSize int, matrixRPath string, MappingList []int64, encryptedPatients []*ckks.Ciphertext, nbrGoroutines, nbrTargetSnps, nbrTargetSnpsInBatch int) (predictionsSave [][]*ckks.Ciphertext) {

	modelParams := &lib.Params
	modelParams.Gen()

	// batch parameters (parallel on target positions)
	numberOfBatches, lastBatchSize := lib.NbrBatchAndLastBatchSize(nbrTargetSnps, nbrTargetSnpsInBatch)
	log.Print("[Server] number of batches: ", numberOfBatches, " and last batch size ", lastBatchSize)

	//time1 := time.Now()

	// get matrix R
	// MatrixRs: original form, no transformation
	MatrixRs := ReadCoefficients(windowSize, matrixRPath, len(lib.MatrixNames), nbrTargetSnps, nbrGoroutines)

	time2 := time.Now()
	//log.Printf("[Server] read model done in %f s\n", time2.Sub(time1).Seconds())
	//lib.PrintMemUsage()

	// prediction *********************

	// predictions[ng][i][k] batch ng, matrixRi, target k (one ciphertext, for all patients)
	predictions := make([][][]*ckks.Ciphertext, numberOfBatches) // each batch contains lib.model.nbrTargetSnps

	predictionTimings := make([]time.Duration, nbrGoroutines)
	totalTimings := make([]time.Duration, nbrGoroutines)

	var wg sync.WaitGroup
	wg.Add(nbrGoroutines)
	workPerGoRoutine := int(math.Ceil(float64(numberOfBatches) / float64(nbrGoroutines)))
	for ng := 0; ng < nbrGoroutines; ng++ {

		start := ng * workPerGoRoutine
		finish := (ng + 1) * workPerGoRoutine

		if ng == nbrGoroutines-1 {
			finish = numberOfBatches
		}

		go func(start, finish, ng int) {
			predictor := NewPredictor(modelParams)

			var predTime time.Duration
			var mappingList []int64
			var batchSize int

			for p := start; p < finish; p++ {
				predictions[p] = make([][]*ckks.Ciphertext, 3)

				for i := 0; i < 3; i++ {

					totalTime1 := time.Now()

					// Each batch has nbrTargetSnps
					// Predict returns a bunch of target positions in batch
					if p == numberOfBatches-1 {
						mappingList = MappingList[p*nbrTargetSnpsInBatch : p*nbrTargetSnpsInBatch+lastBatchSize]
						batchSize = lastBatchSize
					} else {
						mappingList = MappingList[p*nbrTargetSnpsInBatch : (p+1)*nbrTargetSnpsInBatch]
						batchSize = nbrTargetSnpsInBatch
					}

					predictions[p][i], predTime = predictor.Predict(MatrixRs[i], mappingList, encryptedPatients, windowSize, nbrTargetSnpsInBatch, batchSize, p)

					totalTime2 := time.Now()
					predictionTimings[ng] = predictionTimings[ng] + predTime
					totalTimings[ng] = totalTimings[ng] + totalTime2.Sub(totalTime1)
				}
			}
			wg.Done()
		}(start, finish, ng)
	}
	wg.Wait()

	// ***********  re-packing of the predictions ***********
	// First changes predictions[p][i][k] (batch p, i=0,1,2, k is target idx in batch p)
	// to predictionsSave[i][target] (i=0,1,2; target from 0 to 80882)
	predictionsSave = make([][]*ckks.Ciphertext, 3)
	for i := 0; i < 3; i++ {
		predictionsSave[i] = make([]*ckks.Ciphertext, nbrTargetSnps) // i=0,1,2
	}

	for p := 0; p < numberOfBatches; p++ {
		for i := 0; i < 3; i++ {

			if p == numberOfBatches-1 {

				for k := 0; k < lastBatchSize; k++ {
					predictionsSave[i][p*nbrTargetSnpsInBatch+k] = predictions[p][i][k]
				}

			} else {

				for k := 0; k < nbrTargetSnpsInBatch; k++ {
					predictionsSave[i][p*nbrTargetSnpsInBatch+k] = predictions[p][i][k]
				}
			}
		}
	}

	maxPrediction := time.Duration(0)
	sumPred := time.Duration(0)
	for i := 0; i < nbrGoroutines; i++ {
		sumPred = sumPred + predictionTimings[i]
		if predictionTimings[i] > maxPrediction {
			maxPrediction = predictionTimings[i]
		}
	}

	predictionAverage := float64(sumPred.Seconds()) / float64(nbrGoroutines)

	//log.Println("PREDICTION ALL ", predictionTimings)
	log.Println("PREDICTION AVG ", predictionAverage, " AND MAX ", maxPrediction)

	time3 := time.Now()
	log.Printf("[Server] prediction done in %f s\n", time3.Sub(time2).Seconds())

	return
}
