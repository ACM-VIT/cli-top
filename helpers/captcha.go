package helpers

import (
	"bytes"
	"cli-top/debug"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	"math"
	"os"
	"strings"
)

func preImg(img [][]int) [][]int {
	avg := 0
	pixels := 0
	for _, row := range img {
		for _, f := range row {
			avg += f
			pixels++
		}
	}
	if pixels == 0 {
		return nil
	}
	avg /= pixels

	bits := make([][]int, len(img))
	for i := range img {
		bits[i] = make([]int, len(img[i]))
		for j := range img[i] {
			if img[i][j] > avg {
				bits[i][j] = 1
			} else {
				bits[i][j] = 0
			}
		}
	}
	return bits
}

func saturation(d []uint8) [][][]int {
	if len(d) != 40*200*4 {
		return nil
	}
	saturate := make([]int, len(d)/4)
	for i := 0; i < len(d); i += 4 {
		min := uint8(math.Min(float64(d[i]), math.Min(float64(d[i+1]), float64(d[i+2]))))
		max := uint8(math.Max(float64(d[i]), math.Max(float64(d[i+1]), float64(d[i+2]))))
		if max != 0 {
			saturate[i/4] = int(math.Round((float64(max-min) * 255) / float64(max)))
		}
	}

	img := make([][]int, 40)
	for i := 0; i < 40; i++ {
		img[i] = make([]int, 200)
		for j := 0; j < 200; j++ {
			img[i][j] = saturate[i*200+j]
		}
	}

	bls := make([][][]int, 6)
	for i := 0; i < 6; i++ {
		x1 := (i+1)*25 + 2
		y1 := 7 + 5*(i%2) + 1
		x2 := (i+2)*25 + 1
		y2 := 35 - 5*((i+1)%2)
		bls[i] = copySlice(img[y1:y2], func(slice []int) []int { return slice[x1:x2] })
	}

	return bls
}

func copySlice(src [][]int, transform func([]int) []int) [][]int {
	dst := make([][]int, len(src))
	for i, v := range src {
		dst[i] = transform(v)
	}
	return dst
}

func flatten(arr [][]int) []int {
	bits := make([]int, len(arr)*len(arr[0]))
	for i := range arr {
		for j := range arr[i] {
			bits[i*len(arr[0])+j] = arr[i][j]
		}
	}
	return bits
}

func matMul(a [][]int, b [][]float32) []float32 {
	x, z, y := len(a), len(a[0]), len(b[0])
	product := make([][]float32, x)
	for p := 0; p < x; p++ {
		product[p] = make([]float32, y)
	}
	for i := 0; i < x; i++ {
		for j := 0; j < y; j++ {
			for k := 0; k < z; k++ {
				product[i][j] += float32(a[i][k]) * b[k][j]
			}
		}
	}
	return flattenFloat32(product)
}

func matAdd(a []float32, b []float32) []float32 {
	x := len(a)
	c := make([]float32, x)
	for i := 0; i < x; i++ {
		c[i] = a[i] + b[i]
	}
	return c
}

func maxSoft(a []float32) []float32 {
	if len(a) == 0 {
		return nil
	}
	maxValue := a[0]
	for _, value := range a[1:] {
		if value > maxValue {
			maxValue = value
		}
	}
	n := make([]float32, len(a))
	s := float32(0)
	for i, value := range a {
		n[i] = float32(math.Exp(float64(value - maxValue)))
		s += n[i]
	}
	for i := range n {
		n[i] /= s
	}
	return n
}

func flattenFloat32(arr [][]float32) []float32 {
	var flat []float32
	for _, row := range arr {
		flat = append(flat, row...)
	}
	return flat
}

func argmax(slice []float32) int {
	if len(slice) == 0 {
		return -1
	}
	maxIndex := 0
	for i := 1; i < len(slice); i++ {
		if slice[i] > slice[maxIndex] {
			maxIndex = i
		}
	}
	return maxIndex
}

func SolveCaptcha(imageURL string) string {
	labelTxt := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	killSwitch := CheckKillSwitch()
	if killSwitch == 2 {
		return "disabled"
	}
	if !strings.HasPrefix(imageURL, "data:image/jpeg;base64,") {
		fmt.Println("Unsupported URL scheme")
		return ""
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(imageURL, "data:image/jpeg;base64,"))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error decoding captcha:", err)
		}
		return ""
	}
	if killSwitch == 1 {
		StopHeadlineForOutput()
		if err := os.WriteFile("captcha.jpg", data, 0o600); err != nil {
			if debug.Debug {
				fmt.Println("Error saving captcha:", err)
			}
			return ""
		}
		fmt.Println("Captcha auto-solver has been disabled. \nPlease manually solve captcha.jpg and answer here:")
		var captcha string
		fmt.Scanln(&captcha)
		return strings.TrimSpace(captcha)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error decoding captcha image:", err)
		}
		return ""
	}
	bounds := img.Bounds()
	if bounds.Dx() != 200 || bounds.Dy() != 40 {
		if debug.Debug {
			fmt.Printf("Unexpected captcha dimensions: %dx%d\n", bounds.Dx(), bounds.Dy())
		}
		return ""
	}
	rgba := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			rgba.Set(x, y, img.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}

	blocks := saturation(rgba.Pix)
	if len(blocks) != 6 {
		return ""
	}
	var resultText strings.Builder
	for _, block := range blocks {
		flatBlock := flatten(preImg(block))
		result := maxSoft(matAdd(matMul([][]int{flatBlock}, weights), biases))
		maxIndex := argmax(result)
		if maxIndex < 0 || maxIndex >= len(labelTxt) {
			return ""
		}
		resultText.WriteByte(labelTxt[maxIndex])
	}

	result := resultText.String()
	if debug.Debug {
		fmt.Println("(Helper - Captcha):", result)
	}
	return result
}
