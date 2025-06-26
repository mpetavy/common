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

func IsEncrypted(password string) bool {
	return strings.HasPrefix(password, SECRET_PREFIX)
}

func DecryptString(key []byte, txt string) (string, error) {
	if IsEncrypted(txt) {
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
	if !IsEncrypted(txt) {
		s, err := Encrypt(key, []byte(txt))

		return SECRET_PREFIX + base64.StdEncoding.EncodeToString(s), err
	} else {
		return txt, nil
	}
}

func Secret(txt string, secret ...string) (string, error) {
	key := ""

	if len(secret) == 1 {
		key = secret[0]

		DebugFunc("using flag")
	}

	if key == "" {
		for _, file := range []string{"secretkey", "secretkey.txt", ".secretkey", ".secretkey.txt"} {
			if FileExists(file) {
				DebugFunc("using file: %s", file)

				ba, err := os.ReadFile(file)
				if err == nil {
					key = strings.TrimSpace(string(ba))

					break
				}
			}
		}
	}

	if key == "" {
		for _, env := range []string{FlagNameAsEnvName("secretkey"), "SECRETKEY", "secretkey"} {
			key = os.Getenv(env)
			if key != "" {
				DebugFunc("using ENV: %s", env)

				break
			}
		}
	}

	if key == "" {
		return "", fmt.Errorf("SECRETKEY is not defined")
	}

	// descramble
	key = ScrambleString(strings.TrimSpace(key))

	if IsEncrypted(txt) {
		m, err := DecryptString([]byte(key), txt)
		if Error(err) {
			return "", err
		}

		return m, nil
	} else {
		m, err := EncryptString([]byte(key), txt)
		if Error(err) {
			return "", err
		}

		return m, nil
	}
}
