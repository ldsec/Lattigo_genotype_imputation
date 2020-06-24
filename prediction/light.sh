#!/bin/bash

echo "****** Generate csv result files and evaluate micro AUC (for window size 64) ******"

nbrAll=1004
nbrAFR=272
nbrAMR=135
nbrEUR=210
nbrTargets=80882
w=64

./KeyGen

echo "- Accuracy Results for ALL -"
make enc REFDATAPATH=prediction_data/original_data/tag_testing_trimed.txt NBRGOROUTINES=8 BATCHSIZE=2023
make pred WINDOW=${w} NBRGOROUTINES=2 BATCHSIZE=40441 NBRTARGETSNPS=$nbrTargets NBRPATIENTS=$nbrAll POPTYPE=ALL
make dec REFDATAPATH=prediction_data/original_data/tag_testing_trimed.txt NBRGOROUTINES=2 BATCHSIZE=40441 NBRTARGETSNPS=$nbrTargets
cd results
python3 transform.py -i ypred.binary -t ../prediction_data/original_data/answer_targets_80882.txt -c ALL_window$w -o curve -p $nbrAll
cd ..

echo "- Population Stratification -"

echo "*** AFR ***"
make enc REFDATAPATH=prediction_data/original_data/tag_testing_AFR_expa.txt NBRGOROUTINES=8 BATCHSIZE=2023
make pred WINDOW=${w} NBRGOROUTINES=2 BATCHSIZE=40441 NBRTARGETSNPS=$nbrTargets NBRPATIENTS=$nbrAFR POPTYPE=AFR
make dec REFDATAPATH=prediction_data/original_data/tag_testing_AFR_expa.txt NBRGOROUTINES=2 BATCHSIZE=40441 NBRTARGETSNPS=$nbrTargets
cd results
python3 transform.py -i ypred.binary -t ../prediction_data/original_data/answer_AFR_expa.txt -o curve -c AFR_window$w -p $nbrAFR
cd ..

echo "*** AMR ***"
make enc REFDATAPATH=prediction_data/original_data/tag_testing_AMR_expa.txt NBRGOROUTINES=8 BATCHSIZE=2023
make pred WINDOW=${w} NBRGOROUTINES=2 BATCHSIZE=40441 NBRTARGETSNPS=$nbrTargets NBRPATIENTS=$nbrAMR POPTYPE=AMR
make dec REFDATAPATH=prediction_data/original_data/tag_testing_AMR_expa.txt NBRGOROUTINES=2 BATCHSIZE=40441 NBRTARGETSNPS=$nbrTargets
cd results
python3 transform.py -i ypred.binary -t ../prediction_data/original_data/answer_AMR_expa.txt -o curve -c AMR_window$w -p $nbrAMR
cd ..

echo "*** EUR ***"
make enc REFDATAPATH=prediction_data/original_data/tag_testing_EUR_expa.txt NBRGOROUTINES=8 BATCHSIZE=2023
make pred WINDOW=${w} NBRGOROUTINES=2 BATCHSIZE=40441 NBRTARGETSNPS=$nbrTargets NBRPATIENTS=$nbrAMR POPTYPE=EUR
make dec REFDATAPATH=prediction_data/original_data/tag_testing_EUR_expa.txt NBRGOROUTINES=2 BATCHSIZE=40441 NBRTARGETSNPS=$nbrTargets
cd results
python3 transform.py -i ypred.binary -t ../prediction_data/original_data/answer_EUR_expa.txt -o curve -c EUR_window$w -p $nbrEUR
cd ..

