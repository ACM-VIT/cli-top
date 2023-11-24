package login

import (
	"encoding/json"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
)

const (
	Height = 40
	Width  = 200
)

func importBitmaps(filepath string) map[byte][][]uint8 {
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

func captchaParse(imgarr [][]uint8) string {
	bitmaps := importBitmaps("bitmaps.json")
	var captcha string

	for x := 1; x < 44; x++ {
		for y := 1; y < 179; y++ {
			condition1 := imgarr[x][y-1] == 255 && imgarr[x][y] == 0 && imgarr[x][y+1] == 255
			condition2 := imgarr[x-1][y] == 255 && imgarr[x][y] == 0 && imgarr[x+1][y] == 255
			condition3 := imgarr[x][y] != 255 && imgarr[x][y] != 0

			if condition1 || condition2 || condition3 {
				imgarr[x][y] = 255
			}
		}
	}

	for j := 30; j < 181; j += 30 {
		var matches []struct {
			perc float64
			ch   byte
		}

		chars := "123456789ABCDEFGHIJKLMNPQRSTUVWXYZ"
		for i := 0; i < len(chars); i++ {
			var match, black float64
			ch := chars[i]
			mask := bitmaps[ch]

			for x := 0; x < 32; x++ {
				for y := 0; y < 30; y++ {
					y1 := y + j - 30
					x1 := x + 12
					if imgarr[x1][y1] == mask[x][y] && mask[x][y] == 0 {
						match += 1
					}
					if mask[x][y] == 0 {
						black += 1
					}
				}
			}

			perc := match / black
			matches = append(matches, struct {
				perc float64
				ch   byte
			}{perc, ch})
		}

		var maxMatch struct {
			perc float64
			ch   byte
		}
		for _, m := range matches {
			if m.perc > maxMatch.perc {
				maxMatch = m
			}
		}

		captcha += string(maxMatch.ch)
	}

	return captcha
}

func uriToImgData(uri string) ([][]uint8, error) {
	file, err := os.Open(uri)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	var imgData [][]uint8
	for y := 0; y < height; y++ {
		var row []uint8
		for x := 0; x < width; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			gray := float64(r) * 0.299
			row = append(row, uint8(gray))
		}
		imgData = append(imgData, row)
	}

	return imgData, nil
}

func fillCaptcha(imgB64 string) string {
	uri := imgB64
	imgData, err := uriToImgData(uri)
	if err != nil {
		log.Fatal(err)
	}

	var arr []uint8
	var newArr [][]uint8
	for i := 0; i < len(imgData); i += 4 {
		var gval float64
		for _, v := range imgData[i] {
			gval += float64(v)
		}
		gval /= 24 * 22
		arr = append(arr, uint8(gval))
	}
	for len(arr) > 0 {
		newArr = append(newArr, arr[:180])
		arr = arr[180:]
	}

	return captchaParse(newArr)
}

func getCaptcha(src string) string {
	result := fillCaptcha(src)
	// println(result)
	return result
}
