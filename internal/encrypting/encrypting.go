package encrypting

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

type encrypting struct {
	c cipher.AEAD
}

func (e *encrypting) Encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, e.c.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	cipherText := e.c.Seal(nonce, nonce, data, nil)

	return cipherText, nil
}

func (e *encrypting) Decrypt(data []byte) ([]byte, error) {
	nonceSize := e.c.NonceSize()
	nonce, cipherText := data[:nonceSize], data[nonceSize:]

	plainText, err := e.c.Open(nil, nonce, cipherText, nil)

	if err != nil {
		if strings.Contains(err.Error(), "authentication") {
			return nil, errors.New("wrong password")
		}
		return nil, err
	}

	return plainText, nil
}

type EncryptDecrypting interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}

func createHash(key string) string {
	h := md5.New()
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}

func New(passphrase string) (EncryptDecrypting, error) {
	block, err := aes.NewCipher([]byte(createHash(passphrase)))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &encrypting{
		c: gcm,
	}, nil
}