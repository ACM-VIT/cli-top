package helpers

import (
	"cli-top/debug"
	types "cli-top/types"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"os"
	"sort"
	"strings"
)

func preImg(img [][]int) [][]int {
	avg := 0
	for _, row := range img {
		for _, f := range row {
			avg += f
		}
	}
	avg /= 24 * 22

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
	saturate := make([]int, len(d)/4)
	for i := 0; i < len(d); i += 4 {
		min := uint8(math.Min(float64(d[i]), math.Min(float64(d[i+1]), float64(d[i+2]))))
		max := uint8(math.Max(float64(d[i]), math.Max(float64(d[i+1]), float64(d[i+2]))))
		saturate[i/4] = int(math.Round((float64(max-min) * 255) / float64(max)))
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
	n := append([]float32(nil), a...)
	s := float32(0)
	for _, f := range n {
		s += float32(math.Exp(float64(f)))
	}
	for i := range a {
		n[i] = float32(math.Exp(float64(a[i]))) / s
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
	var maxValue float32
	kvs := make([]types.Kv, len(slice))
	for i, v := range slice {
		kvs[i] = types.Kv{Key: i, Value: v}
		if i == 0 || v > maxValue {
			maxValue = v
		}
	}
	sort.Slice(kvs, func(i, j int) bool {
		return kvs[i].Value > kvs[j].Value
	})
	return kvs[0].Key
}

func SolveCaptcha(imageURL string) string {
	labelTxt := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	killSwitch := CheckKillSwitch()

	if strings.HasPrefix(imageURL, "data:image/jpeg;base64,") {
		base64Data := strings.TrimPrefix(imageURL, "data:image/jpeg;base64,")
		data, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil && debug.Debug {
			fmt.Println("Error decoding base64:", err)
			return ""
		}

		img, _, err := image.Decode(strings.NewReader(string(data)))
		if err != nil && debug.Debug {
			fmt.Println(err)
		}
		// Save the image to a file (optional)
		outFile, err := os.Create("captcha.jpg")
		if err != nil && debug.Debug {
			fmt.Println(err)
		}
		defer outFile.Close()

		err = jpeg.Encode(outFile, img, nil)
		if err != nil && debug.Debug {
			fmt.Println(err)
		}

		var captcha string
		if killSwitch == 1 {
			fmt.Println("Captcha auto-solver has been disabled. \nPlease manually solve the captcha and answer here:")
			fmt.Scanln(&captcha)
		} else {
			err = os.Remove("captcha.jpg")
			if err != nil && debug.Debug {
				fmt.Println(err)
			}
		}

		bounds := img.Bounds()
		rgba := image.NewRGBA(bounds)
		for y := 0; y < bounds.Dy(); y++ {
			for x := 0; x < bounds.Dx(); x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}

		pd := rgba.Pix
		bls := saturation(pd)

		var out string
		for i := 0; i < 6; i++ {
			bls[i] = preImg(bls[i])
			flatBls := flatten(bls[i])
			result := matMul([][]int{flatBls}, weights)
			result = matAdd(result, biases)
			result = maxSoft(result)
			maxIndex := argmax(result)
			out += string(labelTxt[maxIndex])
		}

		if debug.Debug {
			fmt.Println("(Helper - Captcha):", out)
		}

		if killSwitch == 1 {
			return captcha
		} else if killSwitch == 2 {
			return "disabled"
		} else {
			return out
		}

	} else {
		fmt.Println("Unsupported URL scheme")
		return ""
	}
}
