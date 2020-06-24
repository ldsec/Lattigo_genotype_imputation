package lib

import (
	"strconv"
)

var KeysPath = "Keys/SecretKey.binary"
var EncDataPath = "temps/"
var PredDataPath = "prediction_data/"
var FreqDataPath = PredDataPath + "frequency.csv"

var ClientParamsPath = EncDataPath + "enc_client_params.binary"
var ClientEncDataPath = EncDataPath + "enc_client_data.binary"
var ClientResDataPath = "results/ypred.binary"

var MaxResultFileSizeMB = 500

var ServerMappingTablePath = PredDataPath + "mapping_table.txt"
var ServerEncParameters = EncDataPath + "enc_pred_parameters.binary"

func MatrixRPath(window string) (path string) {
	return PredDataPath + "/Multinomial_Window" + window + "MatrixR"
}

func ResDataPath(snp, filenbr int) (path string) {
	return EncDataPath + "enc_pred_SNP" + strconv.Itoa(snp) + "_" + strconv.Itoa(filenbr) + ".binary"
}
