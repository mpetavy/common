package common

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

type TLSPackage struct {
	CertificateAsPem, PrivateKeyAsPem []byte
	Config                            tls.Config
}

var (
	tlsCertificate     *string
	tlsKey             *string
	tlsCertificateFile *string
	tlsKeyFile         *string
	muTLS              sync.Mutex
	tlsPack            *TLSPackage
)

const (
	flagCert = "tls.cert"
	flagKey  = "tls.key"
)

func init() {
	tlsCertificate = flag.String(flagCert, "", "TLS server certificate (PEM format)")
	tlsKey = flag.String(flagKey, "", "TLS server private PEM (PEM format)")

	tlsCertificateFile = flag.String("tls.certfile", CleanPath(AppFilename(".cert.pem")), "TLS server certificate file (PEM format)")
	tlsKeyFile = flag.String("tls.keyfile", CleanPath(AppFilename(".cert.key")), "TLS server private key PEM (PEM format)")
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

func TLSConfigFromFile(certFile string, keyFile string) (*TLSPackage, error) {
	DebugFunc("certFile: %s keyFile: %s", certFile, keyFile)

	certOk, err := FileExists(certFile)
	if Error(err) || !certOk {
		return nil, err
	}

	keyOk, err := FileExists(keyFile)
	if Error(err) || !keyOk {
		return nil, err
	}

	var certAsPem []byte
	var keyAsPem []byte

	certAsPem, err = ioutil.ReadFile(certFile)
	if Error(err) {
		return nil, err
	}

	keyAsPem, err = ioutil.ReadFile(keyFile)
	if Error(err) {
		return nil, err
	}

	Debug("generate TLS config from cert file %s and key file %s", certFile, keyFile)

	certificate, err := tls.X509KeyPair([]byte(certAsPem), []byte(keyAsPem))
	if Error(err) {
		return nil, err
	}

	tlsConfig := tls.Config{Certificates: []tls.Certificate{certificate}}
	tlsConfig.Rand = rand.Reader

	return &TLSPackage{
		CertificateAsPem: certAsPem,
		PrivateKeyAsPem:  keyAsPem,
		Config:           tlsConfig,
	}, nil
}

func TLSConfigFromPem(certAsPem []byte, keyAsPem []byte) (*TLSPackage, error) {
	DebugFunc()

	Debug("generate TLS config from given cert and key %s")

	certificate, err := tls.X509KeyPair(certAsPem, keyAsPem)
	if Error(err) {
		return nil, err
	}

	tlsConfig := tls.Config{Certificates: []tls.Certificate{certificate}}
	tlsConfig.Rand = rand.Reader

	return &TLSPackage{
		CertificateAsPem: certAsPem,
		PrivateKeyAsPem:  keyAsPem,
		Config:           tlsConfig,
	}, nil
}

func createTLSPackageByOpenSSL() (*TLSPackage, error) {
	DebugFunc()

	hostname, err := os.Hostname()
	if WarnError(err) {
		hostname = "localhost"
	}

	path, err := exec.LookPath("openssl")

	if path != "" {
		cmd := exec.Command(path, "req", "-new", "-nodes", "-x509", "-out", *tlsCertificateFile, "-keyout", *tlsKeyFile, "-days", "7300", "-subj", "/CN="+hostname)

		err := Watchdog(cmd, time.Second*10)
		if Error(err) {
			return nil, nil
		}

		return TLSConfigFromFile(*tlsCertificateFile, *tlsKeyFile)
	}

	return nil, fmt.Errorf("openssl not available")
}

// https://ericchiang.github.io/post/go-tls/
func CertTemplate() (*x509.Certificate, error) {
	// generate a random serial number (a real cert authority would have some logic behind this)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if Error(err) {
		return nil, err
	}

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{Title()}},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Duration(10) * 365 * 24 * time.Hour),
		BasicConstraintsValid: true,
	}

	return &tmpl, nil
}

func createCert(template, parent *x509.Certificate, pub interface{}, parentPriv interface{}) (
	cert *x509.Certificate, certPEM []byte, err error) {

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pub, parentPriv)
	if err != nil {
		return
	}
	// parse the resulting certificate so we can use it again
	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		return
	}
	// PEM encode the certificate (this is a standard TLS encoding)
	b := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	certPEM = pem.EncodeToMemory(&b)
	return
}

func createTLSPackage() (*TLSPackage, error) {
	// generate a new key-pair
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if Error(err) {
		return nil, err
	}

	rootCertTmpl, err := CertTemplate()
	if Error(err) {
		return nil, err
	}

	ips, err := GetActiveIPs(true)
	if Error(err) {
		return nil, err
	}

	parsedIps := make([]net.IP, 0)
	for _, ip := range ips {
		ip, _, err := net.ParseCIDR(ip)
		if Error(err) {
			return nil, err
		}

		parsedIps = append(parsedIps, ip)
	}

	// describe what the certificate will be used for
	rootCertTmpl.IsCA = true
	rootCertTmpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	rootCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	rootCertTmpl.IPAddresses = parsedIps

	_, rootCertPEM, err := createCert(rootCertTmpl, rootCertTmpl, &rootKey.PublicKey, rootKey)
	if Error(err) {
		return nil, err
	}

	err = ioutil.WriteFile(*tlsCertificateFile, rootCertPEM, DefaultFileMode)
	if Error(err) {
		return nil, err
	}

	// PEM encode the private key
	rootKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rootKey),
	})

	err = ioutil.WriteFile(*tlsKeyFile, rootKeyPEM, DefaultFileMode)
	if Error(err) {
		return nil, err
	}

	return TLSConfigFromFile(*tlsCertificateFile, *tlsKeyFile)
}

func GetTLSPackage(force bool) (*TLSPackage, error) {
	DebugFunc("force: %v", force)

	muTLS.Lock()
	defer muTLS.Unlock()

	var err error

	if tlsPack == nil || force {
		if *tlsCertificateFile != "" && *tlsKeyFile != "" {
			tlsConfig, _ := TLSConfigFromFile(*tlsCertificateFile, *tlsKeyFile)

			if tlsConfig != nil {
				return tlsConfig, nil
			}
		}

		if *tlsCertificate != "" && *tlsKey != "" {
			tlsConfig, _ := TLSConfigFromPem([]byte(*tlsCertificate), []byte(*tlsKey))

			if tlsConfig != nil {
				return tlsConfig, nil
			}
		}

		tlsCert, _ := GetConfiguration().GetFlag(flagCert)
		tlsKey, _ := GetConfiguration().GetFlag(flagKey)

		if tlsCert != "" && tlsKey != "" {
			tlsConfig, _ := TLSConfigFromPem([]byte(tlsCert), []byte(tlsKey))

			if tlsConfig != nil {
				return tlsConfig, nil
			}
		}

		tlsPack, err = createTLSPackage()
	}

	return tlsPack, err
}
