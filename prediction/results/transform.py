#!/usr/bin/python
 
import pandas as pd
import numpy as np
from sklearn.metrics import *
import numpy as np
import matplotlib.pyplot as plt
import sys, getopt
from itertools import islice, cycle

def f(x):
    return np.exp(x)/(1+np.exp(x))
 
# Compute ROC curve and ROC area for each class
def getStats(y_label,y_pred):
    fpr = dict()
    tpr = dict()
    roc_auc = dict()
    n_classes = y_pred.shape[1]
    for i in range(n_classes):
        fpr[i], tpr[i], thres = roc_curve(y_label[:, i], y_pred[:, i])
        roc_auc[i] = auc(fpr[i], tpr[i])
 
    # Compute micro-average ROC curve and ROC area
    fpr["micro"], tpr["micro"], _ = roc_curve(y_label.ravel(), y_pred.ravel())
    roc_auc["micro"] = auc(fpr["micro"], tpr["micro"])
    return fpr, tpr, roc_auc, n_classes
 
# Plot ROC curve
def plotROC(n_classes, roc_auc, fpr, tpr,title, outputfile):
    plt.figure()
    plt.plot(fpr["micro"], tpr["micro"],
             label='micro-average ROC curve (area = {0:0.6f})'
                   ''.format(roc_auc["micro"]))
    for i in range(n_classes):
        plt.plot(fpr[i], tpr[i], label='ROC curve of class {0} (area = {1:0.6f})'
                                       ''.format(i, roc_auc[i]))
 
    plt.plot([0, 1], [0, 1], 'k--')
    plt.xlim([0.0, 1.0])
    plt.ylim([0.0, 1.05])
    plt.xlabel('False Positive Rate')
    plt.ylabel('True Positive Rate')
    plt.title(title)
    plt.legend(loc="lower right")
    plt.savefig(outputfile)
 
 
def main(argv):
    inputfile =''
    outputfile =''
    try:
        opts, args = getopt.getopt(argv,"hi:t:o:c:p:",["ifile=","tfile=","ofile=","csvfile=","patient="])
    except getopt.GetoptError:
        print('python evaluation.py -i <inputfile> -t <targetfile> -o <outputfile> -c <csvfile> -p <patient>')
        sys.exit(2)
    for opt, arg in opts:
        if opt == '-h':
            print('python evaluation.py -i <inputfile> -t <targetfile> -o <outputfile> -c <csvfile> -p <patient>')
            sys.exit()
        elif opt in ("-i", "--ifile"):
            inputfile = arg
        elif opt in ("-t", "--tfile"):
            targetfile = arg
        elif opt in ("-o", "--ofile"):
            outputfile = arg
        elif opt in ("-c", "--csvfile"):
            csvfile = arg
        elif opt in ("-p", "--patient"):
            patient = arg
     
    print('Input file is: ', inputfile)
    print('Target file is: ', targetfile)
    print('Patient number is ', patient)

    # set fixed numbers
    nbr_targets = 80882
    nbr_patients = int(patient) 

    # read in result file in binary format
    data = np.fromfile(inputfile, np.uint16)

    # parse probability for genotype 0,1,2
    list0 = data[0::3]
    list1 = data[1::3]
    list2 = data[2::3]

    # concat columns for genotype 0,1,2
    df0 = pd.DataFrame(list0, columns=['0'])
    df1 = pd.DataFrame(list1, columns=['1'])
    df2 = pd.DataFrame(list2, columns=['2'])
    df = pd.concat([df0, df1], axis=1)
    df = pd.concat([df, df2], axis=1)    

    
    # get average accuracy
    pred_label = pd.to_numeric(df.idxmax(axis=1))
    answer_label = pd.read_csv(targetfile)

    avgacc = sum(pred_label - answer_label['class'] == 0) / answer_label.shape[0]
    print("avgacc =", avgacc)

    # concat the patient indices and position IDs to result file
    target_ids = pd.read_csv("target_list.txt", header=None, dtype='int64')[0].to_list()
    df = df.assign(GenoID=[*islice(cycle(target_ids), nbr_targets * nbr_patients)])
    list = [int(i/nbr_targets) for i in range(0, nbr_targets * nbr_patients)]
    df = df.assign(Patient=list)

    columnsTitles = ['Patient', 'GenoID', '0', '1', '2']
    df = df.reindex(columns=columnsTitles)

    ref = df  #pd.read_csv(inputfile)
    target = pd.read_csv(targetfile)

    print("ref is", inputfile)
    print("target is", targetfile)
    
    # save prediction results in csv format
    ref.to_csv("ypred_" + csvfile + ".csv", sep=',', index=False)
    print("transform to csv file: done.")

    # for reference data
    print(20*"-"+"MicroAUC for reference data"+20*"-")
 
    y_label_ref = pd.get_dummies(target['class'].values).reindex(columns=[0,1,2], fill_value=0).values # answers
    y_pred_ref = ref.iloc[0:, 2:].values # predictions

    print(y_label_ref.shape)
     
    fpr_ref, tpr_ref, roc_auc_ref, n_classes_ref = getStats(y_label_ref, y_pred_ref)
    print('MicroAUC for %d classes ' % n_classes_ref)
    print(roc_auc_ref)
     
    plotROC(n_classes_ref, roc_auc_ref, fpr_ref, tpr_ref,'ROC curves for reference data', outputfile)
 
if __name__ == "__main__":
    main(sys.argv[1:])
