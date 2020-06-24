### Runcommand

##### Build Docker

```
docker build -t epfl_solution .
docker run -it epfl_solution
```

##### Run timings

```
make perf 2>&1 | tee timingsToSendBack.txt
```

##### Run AUC

```
make light 2>&1 | tee aucToSendBackLight.txt // light version to generate table 2 (window size 64)
```



##### Data files

- The plaintext models (coefficients) are in `prediction/prediction_data`
- All the original files for testing (including the population) are in `prediction/prediction_data/original_data`

-  Plaintext models for population stratification are choosed between [trained for all] matrices and [trained for population x] matrices according to the highest prediction results 

##### Result files

- The direct results are save in binary format in `ypred.binary` (for efficiency consideration)
- After running `make auc`, the prediction results are save in the folder `results` with format `ypred_window[4/8/16/32/48/64].csv`, and population stratification results are files with format `ypred_[AFR/AMR/EUR]_window[4/8/16/32/64].csv`

- All the result csv files are in required format as shown in the email

  ```
  Patient,GenoID,0,1,2
  0,17084716,-0.0519483,0.0480828,-0.0553241
  0,17084761,-0.0602524,0.0287036,-0.036127
  0,17085108,0.00387097,-0.0046014,-0.0691283
  0,17086613,0.00317046,-0.00328952,-0.0551729
  ```

- Results that are ready for evalutation (runned by us) are in `done`

- To get a specific result files, run each of the following commands 

  ```
  make build
  ```

  ```
  make run REFDATAPATH=prediction_data/original_data/tag_testing_trimed.txt WINDOW=16 ENCGOROUTINES=4 ENCBATCHSIZE=250 PREDGOROUTINES=4 PREDBATCHSIZE=5000 DECGOROUTINES=4 DECBATCHSIZE=2500 NBRTARGETSNPS=80882 NBRPATIENTS=1004 POPTYPE=ALL
  ```

  - Note
    - REFDATAPATH is input files (tag SNPs)
    - WINDOW is plaintext model type
    - NBRPATIENTS and POPTYPE shoule be consistent, (1004 ALL; 272 AFR; 135 AMR; 210 EUR)

  ```
  python3 transform.py -i ypred.binary -t ../prediction_data/original_data/answer_targets_80882.txt -o curve -c window16 -p 1004
  ```

  - Note 
    - `-i` : input predictions
    - `-t` : answer files
    - `-o` : curve figure
    - `-c` : csv file name
    - `-p ` : number of patients 
