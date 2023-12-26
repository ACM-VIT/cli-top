package helpers

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

func ImportBitmaps(filepath string) map[byte][][]uint8 {
	jsonFile, err := os.Open(filepath) // replace with your json file name
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var data map[byte][][]uint8
	json.Unmarshal(byteValue, &data)

	return data
}
