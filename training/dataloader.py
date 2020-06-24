import pandas as pd
import numpy as np

def findPosInRef(target_pos, df_tag, df_target):
    targetID = df_target.iloc[target_pos, 1]
    
    bound = df_tag.shape[0] - 1

    # find two nearest refID that clamps targetID
    ref_pos = -3
    if df_tag[1][0] > targetID:
        return -1 # most left one
    if df_tag[1][bound] < targetID: # 1044 for data10k, 9745 for data1k
        return -2 # most right one
    for i in range (0, bound): # 1044 for data10k, 9745 for data1k 
        if df_tag[1][i] < targetID and df_tag[1][i+1] > targetID:
            ref_pos = i
            break
    #print("target position in reference:", ref_pos)
    return ref_pos 


class Dataset:
    
    def __init__(self, loci, df_tag, df_target, window):
        self.data, self.label = self.load(loci, df_tag, df_target, window)

    def load(self, loci, df_tag, df_target, window):
        targetPosInRef = findPosInRef(loci, df_tag, df_target)

        df_tag = df_tag.drop(df_tag.index[:4], axis=1)
        df_target = df_target.drop(df_target.index[:4], axis=1) 

        # let df_tag (training data) be 1 2 3 instead of 0 1 2        
        df_tag = df_tag + 1

        bound = df_tag.shape[0] #should be 83072 for the whole dataset

        if targetPosInRef == -3:
            print("error occurred in finding target pos in ref!")
        elif targetPosInRef == -1:
            df_tag = df_tag.iloc[0 : window-1]
        elif targetPosInRef == -2:
            df_tag = df_tag.iloc[bound-window+1 : bound] # 1014:1045 for data10k, 9714:9745 for data1k
        elif targetPosInRef < int(window/2) - 1:
            df_tag = df_tag.iloc[0 : window-1]
        elif targetPosInRef > bound - window/2: # 1045-16 for data10k, 9745-16 for data1k
            df_tag = df_tag.iloc[bound-window+1 : bound] # 1014:1045 for data10k, 9714:9745 for data1k
        else:
            gap = int(window/2)
            df_tag = df_tag.iloc[targetPosInRef-gap+1:targetPosInRef+gap, :]      
        
        data = np.transpose(df_tag.values)
        label = df_target.iloc[loci].values

        return data, label
