package main

import (
	"log"
	"os"
)

// Cleans the temporary files, which includes the keys, encrypted data and results.
func main() {
	cleanFolder("keys/")
	cleanFolder("temps/")
	cleanFolder("results/")
}

func cleanFolder(folderPath string) {

	folder, _ := os.Open(folderPath)
	files, _ := folder.Readdir(0)

	for i := range files {

		name := files[i].Name()

		if name != "do_not_remove.txt" && name != "acc.py" && name != "target_list.txt" && name != "transform.py" {
			if err := os.Remove(folderPath + files[i].Name()); err != nil {
				log.Println(err)
			}
		}
	}
}
