import argparse
import numpy as np
import math

from sklearn.linear_model import RidgeClassifier, SGDClassifier, LogisticRegression, LinearRegression
from sklearn.model_selection import train_test_split
from sklearn import datasets
from sklearn import preprocessing

from dataloader import *



def singleposition_lr_coef(pos, df_tag_train, df_target_train, df_tag_test, df_target_test, window):
    idash_dataset_train = Dataset(pos, df_tag_train, df_target_train, window)
    data_train, label_train = idash_dataset_train.data, idash_dataset_train.label  
    idash_dataset_test = Dataset(pos, df_tag_test, df_target_test, window)
    data_test, label_test = idash_dataset_test.data, idash_dataset_test.label

    vec_one = np.r_[float(1), np.ones(window-1, dtype=float)]
    vec_neg_one = -vec_one
    
    # when targets have only one label, return directly
    if sum(label_train==1) == 0 and sum(label_train==2) == 0:  # only one label 0
        print("only label 0")
        return 1.0, vec_one, vec_neg_one, vec_neg_one 

    if sum(label_train==0) == 0 and sum(label_train==2) == 0:  # only one label 1
        print("only label 1")
        return 1.0, vec_neg_one, vec_one, vec_neg_one

    if sum(label_train==0) == 0 and sum(label_train==1) == 0:  # only one label 2
        print("only label 2")
        return 1.0, vec_neg_one, vec_neg_one, vec_one


    # train for position pos with integer encoding
    clf = LogisticRegression(multi_class='multinomial', max_iter=5000, solver='lbfgs', random_state=1)
    clf.fit(data_train, label_train)

    # predict on test data    
    score = clf.score(data_test, label_test)

    print("score on test set", score)
    
    #print(clf.coef_.shape)
    #print(clf.coef_)

    # when targets have only two labels
    if clf.coef_.shape[0] == 1:
        if sum(label_train==2) == 0:  # label 0 and label 1
            print("label 0 and label 1")
            return score, -np.r_[clf.intercept_[0], clf.coef_[0]], np.r_[clf.intercept_[0], clf.coef_[0]], vec_neg_one
        elif sum(label_train==1) == 0: #label 0 and label 2
            print("label 0 and label 2")
            return score, -np.r_[clf.intercept_[0], clf.coef_[0]], vec_neg_one, np.r_[clf.intercept_[0], clf.coef_[0]]
        elif sum(label_train==0) == 0: # label 1 and label 2
            print("label 1 and label 2")
            return score, vec_neg_one, -np.r_[clf.intercept_[0], clf.coef_[0]], np.r_[clf.intercept_[0], clf.coef_[0]]
    else:  # three labels
        return score, np.r_[clf.intercept_[0], clf.coef_[0]], np.r_[clf.intercept_[1], clf.coef_[1]], np.r_[clf.intercept_[2], clf.coef_[2]]



def main():

    # parse arguments
    parser = argparse.ArgumentParser(description='Hyperparameters')
    parser.add_argument('--window', help='window size')    
    parser.add_argument('--start', help='starting target position')
    parser.add_argument('--end', help='ending target position(excluded)')
    args = parser.parse_args()
    window = int(args.window)
    start = int(args.start)
    end = int(args.end)

    result = []
    df_tag_train = pd.read_csv("tag_training.txt", sep='\t', header=None)
    df_target_train = pd.read_csv("target_geno_extracted_training.txt", sep='\t', header=None)
    df_tag_test = pd.read_csv("tag_testing.txt", sep='\t', header=None)
    df_target_test = pd.read_csv("target_geno_extracted_testing.txt", sep='\t', header=None)

    # generate coefficient matrix R, three in total for 0 1 2
    print("generating coefficient column at position", start)
    score, MatrixR0, MatrixR1, MatrixR2 = singleposition_lr_coef(start, df_tag_train, df_target_train, df_tag_test, df_target_test, window)
    result.append(score)

    for pos in range (start+1, end):
        print("generating coefficient column at position", pos)
        score, coef_col0, coef_col1, coef_col2 = singleposition_lr_coef(pos, df_tag_train, df_target_train, df_tag_test, df_target_test, window) 
        
        result.append(score)
        MatrixR0 = np.c_[MatrixR0, coef_col0]
        MatrixR1 = np.c_[MatrixR1, coef_col1]
        MatrixR2 = np.c_[MatrixR2, coef_col2]    
 

    np.savetxt("Multinomial_IntCode_" + args.start + "_" + args.end + "_Window"+ args.window + "MatrixR0.csv", MatrixR0, delimiter=' ')
    np.savetxt("Multinomial_IntCode_" + args.start + "_" + args.end + "_Window"+ args.window + "MatrixR1.csv", MatrixR1, delimiter=' ')
    np.savetxt("Multinomial_IntCode_" + args.start + "_" + args.end + "_Window"+ args.window + "MatrixR2.csv", MatrixR2, delimiter=' ')

    print("%.6f (+/- %.2f)" % (np.mean(result), np.std(result)))

    result_list = np.array(result)
    np.savetxt("Multinomial_IntCode_" + args.start + "_" + args.end + "_AccuracyList.txt", result_list)

if __name__ == '__main__':
    main() 
