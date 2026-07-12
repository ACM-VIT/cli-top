package cmd

import "testing"

func TestPasswordEncryptionRoundTrip(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := encryptPassword("correct horse battery staple", key)
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "correct horse battery staple" {
		t.Fatal("password was stored as plaintext")
	}
	decrypted, err := decryptPassword(encrypted, key)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "correct horse battery staple" {
		t.Fatalf("decrypted password = %q", decrypted)
	}
}

func TestPasswordEncryptionRejectsInvalidInput(t *testing.T) {
	if _, err := encryptPassword("secret", "short"); err == nil {
		t.Fatal("encryptPassword accepted an invalid key")
	}
	if _, err := decryptPassword("not-base64", "01234567890123456789012345678901"); err == nil {
		t.Fatal("decryptPassword accepted invalid ciphertext")
	}
}
