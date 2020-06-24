# Basic go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get


# Binary names
BINARY_NAME=idash2019
BINARY_UNIX=$(BINARY_NAME)_unix


build:
	${GOBUILD} KeyGen.go
	${GOBUILD} ClientEnc.go
	${GOBUILD} ServerPred.go
	${GOBUILD} ClientDec.go


key:
	./KeyGen 
enc:
	./ClientEnc ${REFDATAPATH} ${NBRGOROUTINES} ${BATCHSIZE}
pred:
	./ServerPred ${WINDOW} ${NBRGOROUTINES} ${BATCHSIZE} ${NBRTARGETSNPS} ${NBRPATIENTS} ${POPTYPE}
dec:
	./ClientDec ${REFDATAPATH} ${NBRGOROUTINES} ${BATCHSIZE}

run:
	./KeyGen 
	./ClientEnc ${REFDATAPATH} ${ENCGOROUTINES} ${ENCBATCHSIZE}
	./ServerPred ${WINDOW} ${PREDGOROUTINES} ${PREDBATCHSIZE} ${NBRTARGETSNPS} ${NBRPATIENTS} ${POPTYPE}
	./ClientDec ${REFDATAPATH} ${DECGOROUTINES} ${DECBATCHSIZE}
	#python3 results/evaluation.py -i results/ypred.binary -t prediction_data/original_data/answer_targets_80882.txt -o results/curve

clean:
	${GOBUILD} Clean.go
	./Clean
	$(GOCLEAN)

perf:
	${GOBUILD} KeyGen.go
	${GOBUILD} ClientEnc.go
	${GOBUILD} ServerPred.go
	${GOBUILD} ClientDec.go
	# timing
	sh timings.sh

light:
	${GOBUILD} KeyGen.go
	${GOBUILD} ClientEnc.go
	${GOBUILD} ServerPred.go
	${GOBUILD} ClientDec.go
	# light AUC evaluation
	sh light.sh
