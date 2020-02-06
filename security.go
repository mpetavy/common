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
	"github.com/grantae/certinfo"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"software.sslmate.com/src/go-pkcs12"
	"strings"
	"sync"
	"time"
)

// https://ericchiang.github.io/post/go-tls/
// https://blog.kowalczyk.info/article/Jl3G/https-for-free-in-go-with-little-help-of-lets-encrypt.html
// https://stackoverflow.com/questions/13555085/save-and-load-crypto-rsa-privatekey-to-and-from-the-disk
// https://knowledge.digicert.com/solution/SO25985.html
// https://github.com/SSLMate/go-pkcs12/blob/master/pkcs12.go

type TLSPackage struct {
	CertificateAsPem, PrivateKeyAsPem []byte
	Info                              string
	RootCA                            *x509.CertPool
	Config                            tls.Config
}

var (
	FlagTlsCertificate *string
	FlagTlsKey         *string
	FlagTlsP12File     *string
	muTLS              sync.Mutex
)

const (
	PKCS12_PASSWORD = pkcs12.DefaultPassword

	flagCert = "tls.cert"
	flagKey  = "tls.key"
)

func init() {
	FlagTlsCertificate = flag.String(flagCert, "", "TLS server certificate (PEM format)")
	FlagTlsKey = flag.String(flagKey, "", "TLS server private PEM (PEM format)")

	FlagTlsP12File = flag.String("tls.p12file", CleanPath(AppFilename(".p12")), "TLS PKCS12 certificates & privkey container file (P12 format)")
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

// the priority on entities inside p12 must be honored
// 1st	  private key
// 2nd	  computer certificate
// 3d..n  CA certificates (will be ignored by app)
func TLSConfigFromP12File(p12File string) (*TLSPackage, error) {
	DebugFunc("p12File: %s", p12File)

	ok, err := FileExists(p12File)
	if Error(err) || !ok {
		return nil, err
	}

	ba, err := ioutil.ReadFile(p12File)
	if Error(err) || !ok {
		return nil, err
	}

	return TLSConfigFromP12Buffer(ba)
}

func TLSConfigFromP12Buffer(ba []byte) (*TLSPackage, error) {
	DebugFunc()

	key, cert, caCerts, err := pkcs12.DecodeChain(ba, PKCS12_PASSWORD)
	if Error(err) {
		return nil, err
	}

	_, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("Expected RSA private key type")
	}

	keyAsPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key.(*rsa.PrivateKey)),
		},
	)

	certAsPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(certAsPem)

	certificate, err := tls.X509KeyPair([]byte(certAsPem), []byte(keyAsPem))
	if Error(err) {
		return nil, err
	}

	tlsConfig := tls.Config{Certificates: []tls.Certificate{certificate}}
	tlsConfig.Rand = rand.Reader

	certInfos, err := CertificateInfoFromX509(append(caCerts, cert))
	if Error(err) {
		return nil, err
	}

	return &TLSPackage{
		CertificateAsPem: certAsPem,
		PrivateKeyAsPem:  keyAsPem,
		Info:             certInfos,
		RootCA:           caCertPool,
		Config:           tlsConfig,
	}, nil
}

func TLSConfigFromPem(certAsPem []byte, keyAsPem []byte) (*TLSPackage, error) {
	DebugFunc("generate TLS config from given cert and key flags")

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

// https://ericchiang.github.io/post/go-tls/
func CertTemplate() (*x509.Certificate, error) {
	hostname, err := os.Hostname()
	if Error(err) {
		return nil, err
	}

	// generate a random serial number (a real cert authority would have some logic behind this)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if Error(err) {
		return nil, err
	}

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{TitleVersion(true, true, true)}},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Duration(10) * 365 * 24 * time.Hour),
		DNSNames:              []string{hostname},
		BasicConstraintsValid: true,
	}

	return &tmpl, nil
}

func createCert(template, parent *x509.Certificate, pub interface{}, parentPriv interface{}) (cert *x509.Certificate, certPEM []byte, err error) {
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
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if Error(err) {
		return nil, err
	}

	certTmpl, err := CertTemplate()
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

	certTmpl.IsCA = true
	certTmpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	certTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	certTmpl.IPAddresses = parsedIps

	cert, certPEM, err := createCert(certTmpl, certTmpl, &key.PublicKey, key)
	if Error(err) {
		return nil, err
	}

	//err = ioutil.WriteFile(*tlsCertificateFile, certPEM, DefaultFileMode)
	//if Error(err) {
	//	return nil, err
	//}

	// PEM encode the private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	//err = ioutil.WriteFile(*tlsKeyFile, keyPEM, DefaultFileMode)
	//if Error(err) {
	//	return nil, err
	//}

	pfx, err := pkcs12.Encode(rand.Reader, key, cert, nil, PKCS12_PASSWORD)
	if Error(err) {
		return nil, err
	}

	err = ioutil.WriteFile(*FlagTlsP12File, pfx, DefaultFileMode)
	if Error(err) {
		return nil, err
	}

	return TLSConfigFromPem(certPEM, keyPEM)
}

func GetTLSPackage() (*TLSPackage, error) {
	DebugFunc()

	muTLS.Lock()
	defer muTLS.Unlock()

	var tlsPackage *TLSPackage

	if *FlagTlsP12File != "" {
		tlsPackage, _ = TLSConfigFromP12File(*FlagTlsP12File)

		if tlsPackage != nil {
			return tlsPackage, nil
		}
	}

	if *FlagTlsCertificate != "" && *FlagTlsKey != "" {
		tlsConfig, _ := TLSConfigFromPem([]byte(*FlagTlsCertificate), []byte(*FlagTlsKey))

		if tlsConfig != nil {
			return tlsConfig, nil
		}
	}

	cfg := GetConfiguration()

	if cfg != nil {
		tlsCert, _ := cfg.GetFlag(flagCert)
		tlsKey, _ := cfg.GetFlag(flagKey)

		if tlsCert != "" && tlsKey != "" {
			tlsConfig, _ := TLSConfigFromPem([]byte(tlsCert), []byte(tlsKey))

			if tlsConfig != nil {
				return tlsConfig, nil
			}
		}
	}

	return createTLSPackage()
}

func VerifyP12(p12 []byte, password string) (*x509.Certificate, *rsa.PrivateKey, error) {
	privateKey, cert, err := pkcs12.Decode(p12, password)
	if err != nil {
		return nil, nil, err
	}
	if err := VerifyCertificate(cert); err != nil {
		return nil, nil, err
	}

	priv, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("Expected RSA private key type")
	}

	return cert, priv, nil
}

func VerifyCertificate(cert *x509.Certificate) error {
	_, err := cert.Verify(x509.VerifyOptions{})
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case x509.CertificateInvalidError:
		switch e.Reason {
		case x509.Expired:
			return fmt.Errorf("Certificate has expired or is not yet valid")
		default:
			return err
		}
	case x509.UnknownAuthorityError:
		return nil
	default:
		return err
	}
}

func CertificateInfoFromConnection(con *tls.Conn) (string, error) {
	txt := ""
	for i, cert := range con.ConnectionState().PeerCertificates {
		header := fmt.Sprintf("#%d ", i)
		info := fmt.Sprintf("%s%s\n", header, strings.Repeat("-", 100-len(header)))

		certInfo, err := certinfo.CertificateText(cert)
		if Error(err) {
			continue
		}

		info += fmt.Sprintf("%s\n", certInfo)

		txt += info
	}

	return txt, nil
}

func CertificateInfoFromX509(certs []*x509.Certificate) (string, error) {
	txt := ""
	for i, cert := range certs {
		header := fmt.Sprintf("#%d ", i)
		info := fmt.Sprintf("%s%s\n", header, strings.Repeat("-", 40-len(header)))

		certInfo, err := certinfo.CertificateText(cert)
		if Error(err) {
			continue
		}

		info += fmt.Sprintf("%s\n", certInfo)

		txt += info
	}

	return txt, nil
}
