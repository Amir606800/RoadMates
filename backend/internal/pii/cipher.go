package pii

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

type Cipher struct{ aead cipher.AEAD }

func New(key []byte) (*Cipher, error) {
	if len(key) != 32 {
		return nil, errors.New("PII encryption requires a 32-byte key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("could not initialize PII encryption")
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.New("could not initialize PII encryption")
	}
	return &Cipher{aead}, nil
}

func applicationAAD(userID, applicationID string) []byte {
	return []byte("roadmates:driver-pin:v1:" + userID + ":" + applicationID)
}

// Format: version byte, random 96-bit nonce, authenticated ciphertext. Binding
// the user and application IDs prevents moving a PIN between applications.
func (c *Cipher) EncryptPIN(pin, userID, applicationID string) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, errors.New("could not encrypt personal information")
	}
	envelope := append([]byte{1}, nonce...)
	return c.aead.Seal(envelope, nonce, []byte(pin), applicationAAD(userID, applicationID)), nil
}

func (c *Cipher) DecryptPIN(envelope []byte, userID, applicationID string) (string, error) {
	minimum := 1 + c.aead.NonceSize() + c.aead.Overhead()
	if len(envelope) < minimum || envelope[0] != 1 {
		return "", errors.New("could not decrypt personal information")
	}
	nonceEnd := 1 + c.aead.NonceSize()
	plain, err := c.aead.Open(nil, envelope[1:nonceEnd], envelope[nonceEnd:], applicationAAD(userID, applicationID))
	if err != nil {
		return "", errors.New("could not decrypt personal information")
	}
	return string(plain), nil
}
