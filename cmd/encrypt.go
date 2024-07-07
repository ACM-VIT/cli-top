package cmd

import (
	"cli-top/debug"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func GenerateAESKey() string {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil && debug.Debug {
		fmt.Println("error generating key")
	}

	keyBase64 := base64.URLEncoding.EncodeToString(key)[:32]
	return keyBase64
}

func encryptPassword(password string, key string) (string, error) {
	if debug.Debug {
		fmt.Println("key", key)
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil && debug.Debug {
		return "", err
	}

	cipherText := make([]byte, aes.BlockSize+len(password))
	iv := cipherText[:aes.BlockSize]
	if _, err := rand.Read(iv); err != nil && debug.Debug {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], []byte(password))

	return base64.URLEncoding.EncodeToString(cipherText), nil
}

func decryptPassword(encryptedPassword string, key string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encryptedPassword)
	if err != nil && debug.Debug {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil && debug.Debug {
		return "", err
	}

	if len(decoded) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	iv := decoded[:aes.BlockSize]
	password := decoded[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(password, password)

	return string(password), nil
}
