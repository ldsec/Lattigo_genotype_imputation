package lib

import (
	"strconv"
)

// KeysPath is a variable pointing to the Keys folder
var KeysPath = "Keys/SecretKey.binary"

// EncDataPath is a variable pointing to the ciphertext(s) folder
var EncDataPath = "temps/"

// PredDataPath is a variable pointing to model(s) folder
var PredDataPath = "prediction_data/"

// ClientParamsPath is a variable pointing to the client's scheme parameters (marshaled)
var ClientParamsPath = EncDataPath + "enc_client_params.binary"

// ClientEncDataPath is a variable pointing to the client encrypted data (marshaled)
var ClientEncDataPath = EncDataPath + "enc_client_data.binary"

// ClientResDataPath is a variable pointing to the client decrypted prediction data (marshaled)
var ClientResDataPath = "results/ypred.binary"

// MaxResultFileSizeMB is a variable setting the maximum file-size of all binary files. If a file
// would larger than this value, it is split into several smaller files.
var MaxResultFileSizeMB = 500

// ServerMappingTablePath is a variable pointing to the model(s)
var ServerMappingTablePath = PredDataPath + "mapping_table.txt"

// ServerEncParameters is a variable pointing to the processed ciphertexts of the server (marshaled)
var ServerEncParameters = EncDataPath + "enc_pred_parameters.binary"

// MatrixRPath returns the path of the model to be run given the window size
func MatrixRPath(window string) (path string) {
	return PredDataPath + "Multinomial_Window" + window + "MatrixR"
}

// ResDataPath returns the path of the result data (encrypted and marshaled) given the target SNP and the file number
func ResDataPath(snp, filenbr int) (path string) {
	return EncDataPath + "enc_pred_SNP" + strconv.Itoa(snp) + "_" + strconv.Itoa(filenbr) + ".binary"
}
