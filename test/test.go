package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type Bitmap struct {
	Ch   string
	Data [][]int
}

func readBitmaps() ([]Bitmap, error) {
	// Read the bitmaps.json file
	fileBytes, err := os.ReadFile("bitmaps.json")
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON data into a slice of Bitmaps
	var bitmaps []Bitmap
	err = json.Unmarshal(fileBytes, &bitmaps)
	if err != nil {
		return nil, err
	}

	return bitmaps, nil
}

func captchaParse(imgarr [][]int) string {

	file, _ := os.ReadFile("bitmaps.json")
	var bitmaps []Bitmap
	_ = json.Unmarshal(file, &bitmaps)

	var captcha string
	for x := 1; x < len(imgarr) && x < 44; x++ {
		for y := 1; y < len(imgarr[x]) && y < 179; y++ {
			condition1 := imgarr[x][y-1] == 255 && imgarr[x][y] == 0 && imgarr[x][y+1] == 255
			condition2 := imgarr[x-1][y] == 255 && imgarr[x][y] == 0 && imgarr[x+1][y] == 255
			condition3 := imgarr[x][y] != 255 && imgarr[x][y] != 0
			if condition1 || condition2 || condition3 {
				imgarr[x][y] = 255
			}
		}
	}

	// The following code for matching with bitmaps and retrieving the captcha string
	// is omitted as it requires additional context and libraries not present in this simplified example

	captcha = ""
	for j := 30; j < 181; j += 30 {
		matches := make(map[string]float64)
		for _, bitmap := range bitmaps {
			match, black := 0, 0
			for x := 0; x < 32; x++ {
				for y := 0; y < 30; y++ {
					y1 := y + j - 30
					x1 := x + 12
					if imgarr[x1][y1] == bitmap.Data[x][y] && bitmap.Data[x][y] == 0 {
						match++
					}
					if bitmap.Data[x][y] == 0 {
						black++
					}
				}
			}
			perc := float64(match) / float64(black)
			matches[bitmap.Ch] = perc
		}

		// Find the character with the highest match percentage
		var maxCh string
		maxPerc := 0.0
		for ch, perc := range matches {
			if perc > maxPerc {
				maxCh = ch
				maxPerc = perc
			}
		}
		captcha += maxCh
	}
	return captcha

}

func main() {
	// Read the image file
	imgFile, err := os.Open("captcha.png")
	if err != nil {
		log.Fatal(err)
	}
	defer imgFile.Close()

	// Read the image data
	imgData, err := io.ReadAll(imgFile)
	if err != nil {
		log.Fatal(err)
	}

	// Encode the image data in base64
	imgBase64 := base64.StdEncoding.EncodeToString(imgData)
	fmt.Println(imgBase64)

	// Read the bitmaps
	// bitmaps, err := readBitmaps()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Assume imgarr is the 2D array of the image data
	var imgarr [][]int

	// Parse the captcha
	captcha := captchaParse(imgarr)

	// Print the decoded captcha
	log.Println("Decoded captcha:", captcha)
}
