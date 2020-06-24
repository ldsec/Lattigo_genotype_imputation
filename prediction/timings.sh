#!/bin/bash

echo "****** time and memory performance for all patients ******"

nbrPatients=1004

echo "- Key Generation -"
./KeyGen

echo "- Encryption -"
for i in 1 2 4 8 16
do
  val=`expr $((16184 / $i))`
  echo "[*Encryption*] GoRoutines $i and batch size $val"
  make enc REFDATAPATH=prediction_data/original_data/tag_testing_trimed.txt NBRGOROUTINES=${i} BATCHSIZE=${val}
done

echo "- Prediction -"
for j in 20000 40000 80000
  do
    echo "**** Nbr of targets $j ****"
    for w in 4 8 16 32 48 64
      do
        echo "**** Window $w ****"
        for i in 1 2 4 8 16
          do
            val=`expr $(($j / $i))`
            echo "-------[*Prediction*] Nbr of targets $j; Window $w; GoRoutines $i; batch size $val;--------"
            make pred WINDOW=${w} NBRGOROUTINES=${i} BATCHSIZE=${val} NBRTARGETSNPS=${j} NBRPATIENTS=$nbrPatients POPTYPE=ALL
          done
      done
  done

echo "- Decryption -"
for j in 20000 40000 80000
  do
    echo "**** Nbr of targets $j ****"
    for i in 1 2 4 8 16
      do
        val=`expr $(($j / $i))`
        batchsizepred=`expr $((3200/(80000 / $j)))`
        echo "[*Decryption*] Nbr of targets $j; GoRoutines $i; batch size $val;"
        make pred WINDOW=16 NBRGOROUTINES=25 BATCHSIZE=${batchsizepred} NBRTARGETSNPS=${j} NBRPATIENTS=$nbrPatients POPTYPE=ALL
        #batchsize=`expr $((1600/(80000 / $j)))`
        make dec REFDATAPATH=prediction_data/original_data/tag_testing_trimed.txt NBRGOROUTINES=${i} BATCHSIZE=${val} NBRTARGETSNPS=${j}
      done
  done


