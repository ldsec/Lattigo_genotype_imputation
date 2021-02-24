import pandas as pd
import argparse

def main():
    parser = argparse.ArgumentParser(description='Hyperparameters')
    parser.add_argument('--answer', help = 'ylabel file path')
    args = parser.parse_args()

    he_pred = pd.read_csv("ypred.csv",sep=',')
    he_pred = he_pred.drop('Subject ID', axis=1)
    he_pred = he_pred.drop('target SNP', axis=1)
    he_pred_label = he_pred.idxmax(axis=1)

    print(he_pred_label.shape)

    file_label = args.answer
    df_label = pd.read_csv(file_label, sep=' ')
    print(df_label.shape)
    cnt = 0
    bound = df_label.shape[0]
    for i in range (0, bound):
        if int(he_pred_label.iloc[i]) == df_label.iloc[i].values:
            cnt = cnt + 1

    print(cnt/bound)

if __name__ == '__main__':
    main()
