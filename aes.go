package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	SECRET_PREFIX = "secret:"
)

func Encrypt(key []byte, message []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if Error(err) {
		return nil, err
	}

	//IV needs to be unique, but doesn't have to be secure.
	//It's common to put it at the beginning of the ciphertext.
	cipherText := make([]byte, aes.BlockSize+len(message))
	iv := cipherText[:aes.BlockSize]
	_, err = io.ReadFull(rand.Reader, iv)
	if Error(err) {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], message)

	return cipherText, nil
}

func Decrypt(key []byte, message []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if Error(err) {
		return nil, err
	}

	if len(message) < aes.BlockSize {
		return nil, errors.New("Ciphertext block size is too short!")
	}

	//IV needs to be unique, but doesn't have to be secure.
	//It's common to put it at the beginning of the ciphertext.
	iv := message[:aes.BlockSize]
	message = message[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(message, message)

	return message, nil
}

func IsStringEnrypted(password string) bool {
	return strings.HasPrefix(password, SECRET_PREFIX)
}

func DecryptString(key []byte, txt string) (string, error) {
	if IsStringEnrypted(txt) {
		txt = txt[len(SECRET_PREFIX):]

		ba, err := base64.StdEncoding.DecodeString(txt)
		if Error(err) {
			return "", err
		}

		s, err := Decrypt(key, ba)
		if Error(err) {
			return "", err
		}

		return string(s), nil
	} else {
		return txt, nil
	}
}

func EncryptString(key []byte, txt string) (string, error) {
	if !IsStringEnrypted(txt) {
		s, err := Encrypt(key, []byte(txt))

		return SECRET_PREFIX + base64.StdEncoding.EncodeToString(s), err
	} else {
		return txt, nil
	}
}

func Secret(txt string) string {
	key := os.Getenv("SECRETKEY")
	if key == "" {
		Panic(fmt.Errorf("SECRETKEY environment variable not set"))
	}

	m, err := DecryptString([]byte(key), txt)
	Panic(err)

	return m
}
