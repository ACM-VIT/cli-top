package helpers

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArgmaxAndSoftmax(t *testing.T) {
	if got := argmax([]float32{-4, 8, 2}); got != 1 {
		t.Fatalf("argmax() = %d, want 1", got)
	}
	if got := argmax(nil); got != -1 {
		t.Fatalf("argmax(nil) = %d, want -1", got)
	}

	values := maxSoft([]float32{10_000, 10_001})
	sum := float64(values[0] + values[1])
	if math.IsNaN(sum) || math.Abs(sum-1) > 1e-6 {
		t.Fatalf("maxSoft() produced %#v", values)
	}
}

func TestSolveCaptchaWithoutDiskRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"test","killSwitch":0}`))
	}))
	defer server.Close()

	originalURL := GetLatestJSONURL()
	SetLatestJSONURL(server.URL)
	defer SetLatestJSONURL(originalURL)

	img := image.NewRGBA(image.Rect(0, 0, 200, 40))
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, img, nil); err != nil {
		t.Fatal(err)
	}
	dataURL := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes())

	if got := SolveCaptcha(dataURL); len(got) != 6 {
		t.Fatalf("SolveCaptcha() = %q, want six characters", got)
	}
}
