package crypto

import (
	"testing"
)

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	// Create encryptor with a fixed key for testing
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	enc, err := NewEncryptorWithKey(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple text",
			plaintext: "hello world",
		},
		{
			name:      "api key",
			plaintext: "sk-1234567890abcdef",
		},
		{
			name:      "unicode text",
			plaintext: "Hello, \u4e16\u754c!",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "long text",
			plaintext: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := enc.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("encrypt failed: %v", err)
			}

			// Decrypt
			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("roundtrip failed: got %q, want %q", decrypted, tt.plaintext)
			}

			// Verify ciphertext is different from plaintext (unless empty)
			if tt.plaintext != "" && ciphertext == tt.plaintext {
				t.Error("ciphertext should be different from plaintext")
			}
		})
	}
}

func TestEncryptor_InvalidKey(t *testing.T) {
	_, err := NewEncryptorWithKey([]byte("short"))
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestEncryptor_InvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := NewEncryptorWithKey(key)

	// Test with invalid base64
	_, err := enc.Decrypt("not-valid-base64!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}

	// Test with valid base64 but invalid ciphertext
	_, err = enc.Decrypt("SGVsbG8gV29ybGQ=") // "Hello World" in base64
	if err != ErrInvalidCiphertext {
		t.Errorf("expected ErrInvalidCiphertext, got %v", err)
	}
}

func TestEncryptor_DifferentCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := NewEncryptorWithKey(key)

	plaintext := "same plaintext"

	// Encrypt twice - should produce different ciphertexts due to random nonce
	ct1, _ := enc.Encrypt(plaintext)
	ct2, _ := enc.Encrypt(plaintext)

	if ct1 == ct2 {
		t.Error("expected different ciphertexts for same plaintext (different nonces)")
	}

	// Both should decrypt to the same plaintext
	pt1, _ := enc.Decrypt(ct1)
	pt2, _ := enc.Decrypt(ct2)

	if pt1 != plaintext || pt2 != plaintext {
		t.Error("both ciphertexts should decrypt to original plaintext")
	}
}
