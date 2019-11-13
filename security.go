package common

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"sync"
	"time"
)

var (
	tlsCertificate     *string
	tlsKey             *string
	TLSCertificateFile *string
	TLSKeyFile         *string
	muTLS              sync.Mutex
	tlsConfig          *tls.Config
)

const (
	flagCert = "tls.cert"
	flagKey  = "tls.key"
)

func init() {
	tlsCertificate = flag.String(flagCert, "", "TLS server certificate (PEM format)")
	tlsKey = flag.String(flagKey, "", "TLS server private PEM (PEM format)")

	TLSCertificateFile = flag.String("tls.certfile", AppFilename(".cert.pem"), "TLS server certificate file (PEM format)")
	TLSKeyFile = flag.String("tls.keyfile", AppFilename(".key.pem"), "TLS server private key PEM (PEM format)")
}

func Rnd(max int) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}

	return int(nBig.Int64())
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func TLSConfigFromFile(certFile string, keyFile string) (*tls.Config, error) {
	DebugFunc("certFile: %s keyFile_ %s", certFile, keyFile)

	certOk, err := FileExists(certFile)
	if Error(err) || !certOk {
		return nil, err
	}

	keyOk, err := FileExists(keyFile)
	if Error(err) || !keyOk {
		return nil, err
	}

	var pemCert []byte
	var pemKey []byte

	pemCert, err = ioutil.ReadFile(certFile)
	if Error(err) {
		return nil, err
	}

	pemKey, err = ioutil.ReadFile(keyFile)
	if Error(err) {
		return nil, err
	}

	Debug("generate TLS config from cert file %s and key file %s", certFile, keyFile)

	cert, err := tls.X509KeyPair([]byte(pemCert), []byte(pemKey))
	if Error(err) {
		return nil, err
	}

	var tlsConfig tls.Config

	tlsConfig = tls.Config{Certificates: []tls.Certificate{cert}}
	tlsConfig.Rand = rand.Reader

	return &tlsConfig, nil
}

func TLSConfigFromPem(pemCert []byte, pemKey []byte) (*tls.Config, error) {
	DebugFunc()

	Debug("generate TLS config from given cert and key %s")

	cert, err := tls.X509KeyPair([]byte(pemCert), []byte(pemKey))
	if Error(err) {
		return nil, err
	}

	var tlsConfig tls.Config

	tlsConfig = tls.Config{Certificates: []tls.Certificate{cert}}
	tlsConfig.Rand = rand.Reader

	return &tlsConfig, nil
}

func createTLSConfig() (*tls.Config, error) {
	DebugFunc()

	hostname, err := os.Hostname()
	if WarnError(err) {
		hostname = "localhost"
	}

	path, err := exec.LookPath("openssl")

	if path != "" {
		cmd := exec.Command(path, "req", "-new", "-nodes", "-x509", "-out", *TLSCertificateFile, "-keyout", *TLSKeyFile, "-days", "7300", "-subj", "/CN="+hostname)

		err := Watchdog(cmd, time.Second*10)
		if Error(err) {
			return nil, nil
		}

		return TLSConfigFromFile(*TLSCertificateFile, *TLSKeyFile)
	}

	return nil, fmt.Errorf("openssl not available")
}

func GetTLSConfig(force bool) (*tls.Config, error) {
	DebugFunc("force: %v", force)

	muTLS.Lock()
	defer muTLS.Unlock()

	var err error

	if tlsConfig == nil || force {
		if *TLSCertificateFile != "" && *TLSKeyFile != "" {
			tlsConfig, _ = TLSConfigFromFile(*TLSCertificateFile, *TLSKeyFile)

			if tlsConfig != nil {
				return tlsConfig, nil
			}
		}

		if *tlsCertificate != "" && *tlsKey != "" {
			tlsConfig, _ = TLSConfigFromPem([]byte(*tlsCertificate), []byte(*tlsKey))

			if tlsConfig != nil {
				return tlsConfig, nil
			}
		}

		tlsCert, _ := GetConfiguration().GetFlag(flagCert)
		tlsKey, _ := GetConfiguration().GetFlag(flagKey)

		if tlsCert != "" && tlsKey != "" {
			tlsConfig, _ = TLSConfigFromPem([]byte(tlsCert), []byte(tlsKey))

			if tlsConfig != nil {
				return tlsConfig, nil
			}
		}

		tlsConfig, err = createTLSConfig()
	}

	return tlsConfig, err
}
