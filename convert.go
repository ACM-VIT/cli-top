package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	file, err := os.Open("input_image.png")
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

	outputFile, err := os.Create("output_image.png")
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	if err := png.Encode(outputFile, grayscale); err != nil {
		panic(err)
	}
}
