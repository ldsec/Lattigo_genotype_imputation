### Homomorphically Encrypted Outsourced Genotype Imputation with Lattigo
This repository hosts the code for the solution submitted by the EPFL-LDS team (https://lds.epfl.ch) to iDash 2019 track II (https://humangenomeprivacy.org/2019), evolved and improved for the article "Secure and Practical Genotype Imputation Methods via Homomorphic Encryption". The code implements a secure genotype imputation service built on top of the Lattigo lattice-based homomorphic encryption library (https://lattigo.epfl.ch), developed by LDS.

Instructions follow below for building the docker and reproducing the timings and performancd figures shown in the aforementioned paper.

##### How to build the Docker

```
docker build -t epfl_solution .
docker run -it epfl_solution
```

##### Running performance evaluation

In order to obtain the execution times of the system for prediction in the test set, you can run

```
make perf 2>&1 | tee timings.txt
```

##### Running accuracy evaluation

In order to obtain the AUC (Area Under Curve) figures for the predictions in the test set, you can run 
```
make light 2>&1 | tee aucLight.txt
```
By default, the script obtains the result for regressions of 64 coefficients (63 neighboring SNPs + 1 intercept variable), and this can be changed in the script to report the accuracy for different regression sizes.


##### Data files
There files included in the repository are structured in different folders, depending on whether they are used for re-training the used models, or for testing their performance and accuracy:

###### Model training
Under the iDash 2019 Track II scenario, the model training happens offline, in a trusted environment. The python scripts required to rerun the training are in the `training` folder. In order to benchmark the sytem with the available training-test dataset, re-training is not needed.

###### Prediction/Imputation (test)
- The plaintext models (regression coefficients) trained on the sample data are in `prediction/prediction_data`
- All the original files for testing (including the population) are in `prediction/prediction_data/original_data`

-  Plaintext models for population stratification are chosen between [trained for all] matrices and [trained for population x] matrices according to the highest prediction results 

##### Result files
Results are saved in the `prediction/results` subfolder.

- When running the system for imputation, the direct results are saved in binary format in `ypred.binary` (for storage efficiency consideration).

- After running `make auc`, the prediction results are saved in the folder `prediction/results`, with format `ypred_window[4/8/16/32/48/64].csv`, and population stratification results are files with name `ypred_[AFR/AMR/EUR]_window[4/8/16/32/64].csv`

- All the mentioned non-binary result files are in a comma-separated-value (CSV) format, with the following structure:

  ```
  Patient,GenoID,0,1,2
  0,17084716,-0.0519483,0.0480828,-0.0553241
  0,17084761,-0.0602524,0.0287036,-0.036127
  0,17085108,0.00387097,-0.0046014,-0.0691283
  0,17086613,0.00317046,-0.00328952,-0.0551729
  ```

- To reproduce the results obtained for the paper "Secure and Practical Genotype Imputation Methods via Homomorphic Encryption" and re-run the encrypted predictions for the test dataset, run the following commands 

  ```
  make build
  ```

  ```
  make run REFDATAPATH=prediction_data/original_data/tag_testing_trimed.txt WINDOW=16 ENCGOROUTINES=4 ENCBATCHSIZE=250 PREDGOROUTINES=4 PREDBATCHSIZE=5000 DECGOROUTINES=4 DECBATCHSIZE=2500 NBRTARGETSNPS=80882 NBRPATIENTS=1004 POPTYPE=ALL
  ```

  - Note
    - REFDATAPATH is the path of input files (tag SNPs)
    - WINDOW is the plaintext model type (equal to the number of neighboring SNPs used for the prediction + 1)
    - NBRPATIENTS and POPTYPE should be consistent with the used dataset (e.g., 1004 for ALL; 272 for AFR; 135 for AMR; 210 for EUR)

  ```
  python3 transform.py -i ypred.binary -t ../prediction_data/original_data/answer_targets_80882.txt -o curve -c window16 -p 1004
  ```

  - The arguments of the transform.py script are the following: 
    - `-i` : input predictions
    - `-t` : answer files
    - `-o` : curve figure
    - `-c` : csv file name
    - `-p ` : number of patients 
