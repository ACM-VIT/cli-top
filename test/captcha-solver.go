package test

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"os"

	"github.com/otiai10/gosseract/v2"
)

func CaptchaSolver() string {
	file, err := os.Open("captcha.jpg")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}

	grayscale := image.NewGray(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			grayscale.Set(x, y, img.At(x, y))
		}
	}

	for y := grayscale.Bounds().Min.Y; y < grayscale.Bounds().Max.Y; y++ {
		for x := grayscale.Bounds().Min.X; x < grayscale.Bounds().Max.X; x++ {
			pixelColor := grayscale.GrayAt(x, y)
			if pixelColor.Y < 128 {
				grayscale.Set(x, y, color.RGBA{0, 0, 0, 255})
			} else {
				grayscale.Set(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}

	outputFile, err := os.Create("output_image.jpg")
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	if err := jpeg.Encode(outputFile, grayscale, nil); err != nil {
		panic(err)
	}

	client := gosseract.NewClient()
	defer client.Close()
	imagePath := "output_image.jpg"
	err = client.SetImage(imagePath)
	if err != nil {
		log.Fatalf("Error setting image: %v", err)
	}
	text, err := client.Text()
	if err != nil {
		log.Fatalf("Error performing OCR: %v", err)
	}
	fmt.Println("OCR Result:")
	fmt.Println(text)
	text = CaptchaParse("output_image.jpg")
	return text
}

func CaptchaParse(path string) string {
	bitmaps := importBitmaps("helpers/bitmaps.json")

	file, err := os.Open(path) // replace with your file name
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}

	bounds := img.Bounds()

	// Example: Check if the image dimensions are not zero
	if bounds.Empty() {
		log.Fatal("Error: Image dimensions are zero")
	}

	width, height := bounds.Max.X, bounds.Max.Y

	fmt.Println(width, height)

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
			condition1 := imgArr[y-1][x] == 255 && imgArr[y][x] == 0 && imgArr[y+1][x] == 255
			condition2 := imgArr[y][x-1] == 255 && imgArr[y][x] == 0 && imgArr[y][x+1] == 255
			condition3 := imgArr[y][x] != 255 && imgArr[y][x] != 0

			if condition1 || condition2 || condition3 {
				// Assuming white is 255 and black is 0 in the grayscale image
				// Set the pixel to white (255) if conditions are met
				imgArr[y][x] = 255
			}
		}
	}

	// Assume imgArr is a 2D slice containing the image data
	// and width and height are the dimensions of the image

	// Create a new image.Gray with the same dimensions as imgArr
	image1 := image.NewGray(image.Rect(0, 0, width, height))

	// Set the pixel values using the values in imgArr
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			image1.Set(x, y, color.Gray{Y: imgArr[y][x]})
		}
	}

	// Create a new file
	file1, err1 := os.Create("output.jpg")
	if err1 != nil {
		panic(err1)
	}
	defer file1.Close()

	// Encode the image.Gray as a JPEG and write it to the file
	err = jpeg.Encode(file1, image1, nil)
	if err != nil {
		fmt.Print("95")
		panic(err)
	}

	// use tesseract here instead of the following logic
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

					if imgArr[y1][x1] == uint8(mask[x][y]) && mask[x][y] == 0 {
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
