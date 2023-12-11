package main

import (
	"fmt"
	"log"

	"github.com/otiai10/gosseract/v2"
)

func main() {
	client := gosseract.NewClient()
	defer client.Close()
	client.SetLanguage("eng")
	imagePath := "output_image.png"
	err := client.SetImage(imagePath)
	if err != nil {
		log.Fatalf("Error setting image: %v", err)
	}
	text, err := client.Text()
	if err != nil {
		log.Fatalf("Error performing OCR: %v", err)
	}
	fmt.Println("OCR Result:")
	fmt.Println(text)
}
