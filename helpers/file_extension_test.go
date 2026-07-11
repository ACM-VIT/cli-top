package helpers

import (
	"archive/zip"
	"bytes"
	"net/http"
	"testing"
)

func TestGetFileExtensionUsesAvailableMetadata(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		headers  http.Header
		body     []byte
		want     string
	}{
		{name: "filename", filename: "notes.PDF", want: ".PDF"},
		{name: "content disposition", headers: http.Header{"Content-Disposition": {`attachment; filename="marks.xlsx"`}}, want: ".xlsx"},
		{name: "pdf signature", body: []byte("%PDF-1.7\n"), want: ".pdf"},
		{name: "unknown", body: []byte("plain text"), want: ".bin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetFileExtension(tt.filename, tt.body, tt.headers); got != tt.want {
				t.Fatalf("GetFileExtension() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetFileExtensionDetectsOOXMLOnce(t *testing.T) {
	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	file, err := writer.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("<document/>")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	if got := GetFileExtension("download", body.Bytes(), nil); got != ".docx" {
		t.Fatalf("GetFileExtension() = %q, want .docx", got)
	}
}
