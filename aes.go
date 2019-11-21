package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

func AESEncrypt(key []byte, message string) (encmess string, err error) {
	plainText := []byte(message)

	block, err := aes.NewCipher(key)
	if Error(err) {
		return "", err
	}

	//IV needs to be unique, but doesn't have to be secure.
	//It's common to put it at the beginning of the ciphertext.
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	_, err = io.ReadFull(rand.Reader, iv)
	if Error(err) {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)

	//returns to base64 encoded string
	encmess = base64.StdEncoding.EncodeToString(cipherText)

	return
}

func AESDecrypt(key []byte, securemess string) (decodedmess string, err error) {
	cipherText, err := base64.StdEncoding.DecodeString(securemess)
	if Error(err) {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if Error(err) {
		return "", err
	}

	if len(cipherText) < aes.BlockSize {
		return "", errors.New("Ciphertext block size is too short!")
	}

	//IV needs to be unique, but doesn't have to be secure.
	//It's common to put it at the beginning of the ciphertext.
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(cipherText, cipherText)

	decodedmess = string(cipherText)

	return
}

func IsStringEnrypted(password string) bool {
	return strings.HasPrefix(password, "enc:")
}

func DecryptString(key []byte, txt string) (string, error) {
	if IsStringEnrypted(txt) {
		return AESDecrypt(key, txt[4:])
	} else {
		return txt, nil
	}
}

func EncryptString(key []byte, txt string) (string, error) {
	if !IsStringEnrypted(txt) {
		password, err := AESEncrypt(key, txt)

		return "enc:" + password, err
	} else {
		return txt, nil
	}
}
