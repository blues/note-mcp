package lib

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

// EncryptedNote represents the encrypted note data in Blues format
type EncryptedNote struct {
	Alg  string `json:"alg"`
	Data string `json:"data"`
	Key  string `json:"key"`
}

// EncryptMessage encrypts a message using ECDH key exchange and AES-256-CBC encryption
func EncryptMessage(publicKeyPEM string, message []byte) (*EncryptedNote, error) {
	// Parse the PEM-encoded public key
	pemBlock, _ := pem.Decode([]byte(publicKeyPEM))
	if pemBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Parse the public key
	publicKeyInterface, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	// Convert to ECDSA public key
	ecdsaPublicKey, ok := publicKeyInterface.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not an ECDSA key")
	}

	// Convert to ECDH public key (P-384 curve)
	curve := ecdh.P384()

	// Generate ephemeral key pair
	ephemeralPrivateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}
	ephemeralPublicKey := ephemeralPrivateKey.PublicKey()

	// Convert the ECDSA public key to ECDH format
	pubKeyBytes := append([]byte{0x04}, append(ecdsaPublicKey.X.Bytes(), ecdsaPublicKey.Y.Bytes()...)...)
	recipientPublicKey, err := curve.NewPublicKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create ECDH public key: %w", err)
	}

	// Perform ECDH key exchange
	sharedSecret, err := ephemeralPrivateKey.ECDH(recipientPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to perform ECDH: %w", err)
	}

	// Derive AES key from shared secret using SHA256
	hash := sha256.Sum256(sharedSecret)
	aesKey := hash[:] // Use full 32-byte hash for AES-256

	// Use zero IV
	iv := make([]byte, aes.BlockSize)

	// Apply PKCS#7 padding
	paddingLength := aes.BlockSize - (len(message) % aes.BlockSize)
	padding := bytes.Repeat([]byte{byte(paddingLength)}, paddingLength)
	paddedMessage := append(message, padding...)

	// Encrypt with AES-256-CBC
	cipherBlock, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	ciphertext := make([]byte, len(paddedMessage))
	mode := cipher.NewCBCEncrypter(cipherBlock, iv)
	mode.CryptBlocks(ciphertext, paddedMessage)

	// Convert ephemeral public key to DER format and base64 encode
	ephemeralKeyDER, err := x509.MarshalPKIXPublicKey(ephemeralPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ephemeral public key: %w", err)
	}

	return &EncryptedNote{
		Alg:  "secp384r1-aes256cbc",
		Data: base64.StdEncoding.EncodeToString(ciphertext), // Only ciphertext, no IV
		Key:  base64.StdEncoding.EncodeToString(ephemeralKeyDER),
	}, nil
}
