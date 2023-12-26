package test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"os"
	"strings"
)

func captchaParse(img image.Image) string {
	bitmaps := importBitmaps("login/bitmaps.json")

	bounds := img.Bounds()

	// Example: Check if the image dimensions are not zero
	if bounds.Empty() {
		log.Fatal("Error: Image dimensions are zero")
	}

	width, height := bounds.Max.X, bounds.Max.Y

	// Initialize a 2D slice to store image data
	imgArr := make([][]uint8, height)
	for i := range imgArr {
		imgArr[i] = make([]uint8, width)
	}

	// Populate imgArr with pixel values
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			grayValue := color.GrayModel.Convert(img.At(x, y)).(color.Gray).Y
			imgArr[y][x] = grayValue
		}
	}

	var captcha string

	// Extracted captcha logic from JavaScript
	for x := 1; x < 44; x++ {
		for y := 1; y < 179; y++ {
			condition1 := img.At(y-1, x).(color.Gray).Y == 255 && img.At(y, x).(color.Gray).Y == 0 && img.At(y+1, x).(color.Gray).Y == 255
			condition2 := img.At(y, x-1).(color.Gray).Y == 255 && img.At(y, x).(color.Gray).Y == 0 && img.At(y, x+1).(color.Gray).Y == 255
			condition3 := img.At(y, x).(color.Gray).Y != 255 && img.At(y, x).(color.Gray).Y != 0

			if condition1 || condition2 || condition3 {
				// Assuming white is 255 and black is 0 in the grayscale image
				// Set the pixel to white (255) if conditions are met
				img.(*image.Gray).Set(y, x, color.Gray{255})
			}
		}
	}

	for j := 30; j < 181; j += 30 {
		matches := make([][2]float64, 0)
		chars := "123456789ABCDEFGHIJKLMNPQRSTUVWXYZ"

		for i := 0; i < len(chars); i++ {
			match := 0.0
			black := 0.0
			ch := chars[i]
			mask := bitmaps[ch]

			for x := 0; x < 32; x++ {
				for y := 0; y < 30; y++ {
					y1 := y + j - 30
					x1 := x + 12

					if img.At(y1, x1).(color.Gray).Y == uint8(mask[x][y]) && mask[x][y] == 0 {
						match += 1
					}

					if mask[x][y] == 0 {
						black += 1
					}
				}
			}

			perc := match / black
			matches = append(matches, [2]float64{perc, float64(ch)})
		}

		maxMatch := matches[0]
		for _, m := range matches {
			if m[0] > maxMatch[0] {
				maxMatch = m
			}
		}

		captcha += string(int(maxMatch[1]))
	}

	if len(imgArr) == 0 || len(imgArr[0]) == 0 {
		log.Fatal("Error: Image array is empty")
	}

	return captcha
}

func convertToGray(src image.Image) *image.Gray {
	bounds := src.Bounds()
	gray := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray.Set(x, y, color.GrayModel.Convert(src.At(x, y)))
		}
	}

	return gray
}

func getCaptcha(src string) string {
	base64String := strings.TrimPrefix(src, "data:image/jpeg;base64,")

	// Decode the base64 string into image data
	imageData, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		log.Fatal("Error decoding base64 string:", err)
	}

	// Decode the image data into an jpeg.Image
	img, err := jpeg.Decode(bytes.NewReader(imageData))
	if err != nil {
		log.Fatal("Error decoding image:", err)
	}

	// Convert the image to grayscale
	grayImg := convertToGray(img)

	// Example: Check if the image is not nil before processing
	if grayImg == nil {
		log.Fatal("Error: Image is nil")
	}

	// Example: Check if the image dimensions are not zero
	bounds := grayImg.Bounds()
	if bounds.Empty() {
		log.Fatal("Error: Image dimensions are zero")
	}

	// Process captcha and print the result
	result := captchaParse(grayImg)
	fmt.Println("Captcha:", result)

	return result
}

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
