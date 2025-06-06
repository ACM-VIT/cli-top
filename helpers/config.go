package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Config represents the structure of config.json
type Config struct {
	Gradient []string `json:"gradient"`
}

var GradColors []string

func ReadConfigJSON() ([]string, error) {
	configPath := filepath.Join(".", "config.json")
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Default to white gradient if not found
			return []string{"#ffffff", "#ffffff", "#ffffff"}, nil
		}
		return nil, err
	}
	defer file.Close()

	// Check if file is empty
	stat, err := file.Stat()
	if err == nil && stat.Size() == 0 {
		return []string{"#ffffff", "#ffffff", "#ffffff"}, nil
	}

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		// If the error is EOF (empty file), return default gradient
		if err == io.EOF {
			return []string{"#ffffff", "#ffffff", "#ffffff"}, nil
		}
		return nil, err
	}

	if len(cfg.Gradient) > 3 {
		fmt.Println("Warning: More than 3 colors specified in gradient. Only the first 3 will be used.")
	}

	// Return only the first 3 colors if more are present, or pad with white if less
	grad := cfg.Gradient
	if len(grad) > 3 {
		grad = grad[:3]
	}
	for len(grad) < 3 {
		grad = append(grad, "#ffffff")
	}
	return grad, nil
}
