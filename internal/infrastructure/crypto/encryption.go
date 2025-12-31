// Package crypto provides encryption utilities for sensitive data.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrInvalidCiphertext is returned when decryption fails due to invalid data.
var ErrInvalidCiphertext = errors.New("invalid ciphertext")

// Encryptor provides encryption and decryption for sensitive data.
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new Encryptor with a machine-derived key.
// The key is derived from a combination of machine-specific identifiers
// to ensure encrypted data can only be decrypted on the same machine.
func NewEncryptor() (*Encryptor, error) {
	key, err := deriveKey()
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}
	return &Encryptor{key: key}, nil
}

// NewEncryptorWithKey creates an Encryptor with a specific key.
// The key should be 32 bytes for AES-256.
func NewEncryptorWithKey(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}
	return &Encryptor{key: key}, nil
}

// Encrypt encrypts plaintext and returns a base64-encoded ciphertext.
// Uses AES-256-GCM for authenticated encryption.
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends the ciphertext to the nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded ciphertext.
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	return string(plaintext), nil
}

// deriveKey derives a 32-byte key from machine-specific identifiers.
// Uses a combination of the hostname and user-specific salt file.
func deriveKey() ([]byte, error) {
	// Get hostname as one component
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}

	// Get or create a salt file in the config directory
	salt, err := getOrCreateSalt()
	if err != nil {
		return nil, err
	}

	// Combine components and hash
	combined := fmt.Sprintf("%s:%s", hostname, string(salt))
	hash := sha256.Sum256([]byte(combined))
	return hash[:], nil
}

// getOrCreateSalt gets or creates a random salt stored in the config directory.
func getOrCreateSalt() ([]byte, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	saltFile := filepath.Join(homeDir, ".skillrunner", ".salt")

	// Try to read existing salt
	salt, err := os.ReadFile(saltFile)
	if err == nil && len(salt) == 32 {
		return salt, nil
	}

	// Create config directory if needed
	configDir := filepath.Dir(saltFile)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate new random salt
	salt = make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Write salt file with restricted permissions
	if err := os.WriteFile(saltFile, salt, 0600); err != nil {
		return nil, fmt.Errorf("failed to write salt file: %w", err)
	}

	return salt, nil
}
