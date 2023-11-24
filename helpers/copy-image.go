package helpers

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func CopyFile(dest string, src string) {
	srcFile, err := os.Open(src) // replace with your source file name
	if err != nil {
		log.Fatal(err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dest) // replace with your destination file name
	if err != nil {
		log.Fatal(err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile) // first parameter is the destination, second is the source
	if err != nil {
		log.Fatal(err)
	}
}

// err := DownloadFile("local_image.png", "http://example.com/image.png") // replace with your local file name and image URL
// if err != nil {
//     panic(err)

func DownloadFile(dest string, src string) error {
	resp, err := http.Get(src)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func DecodeCaptchaFile(dest string, src string) {

	// Remove the "data:image/jpeg;base64," part
	base64Data := strings.Split(src, ",")[1]

	// Decode the base64 data
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Write the data to a file
	err = os.WriteFile(dest, data, 0644)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("Image saved to:", dest)
}
