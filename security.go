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
	"software.sslmate.com/src/go-pkcs12"
	"strings"
	"sync"
	"time"
)

type TlsPackage struct {
	CertificateAsPem, PrivateKeyAsPem []byte
	Certificate                       *x509.Certificate
	PrivateKey                        interface{}
	CaCerts                           []*x509.Certificate
	P12                               []byte
	Info                              string
	Config                            tls.Config
}

var (
	FlagTlsP12File *string
	FlagTlsP12     *string
	muTLS          sync.Mutex
)

const (
	FlagNameTlsP12File = "tls.p12file"
	FlagNameTlsP12     = "tls.p12"
)

func init() {
	FlagTlsP12File = flag.String(FlagNameTlsP12File, "", "TLS PKCS12 certificates & privkey container file (P12 format)")
	FlagTlsP12 = flag.String(FlagNameTlsP12, "", "TLS PKCS12 certificates & privkey container stream (P12,Base64 format)")
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
	DebugFunc()

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
	DebugFunc()

	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

// the priority on entities inside p12 must be honored
// 1st	  private key
// 2nd	  computer certificate
// 3d..n  CA certificates (will be ignored by app)
func TLSConfigFromP12File(p12File string) (*TlsPackage, error) {
	DebugFunc("p12File: %s", p12File)

	ok, err := FileExists(p12File)
	if Error(err) || !ok {
		return nil, err
	}

	ba, err := ioutil.ReadFile(p12File)
	if Error(err) || !ok {
		return nil, err
	}

	return TlsConfigFromP12Buffer(ba)
}

func TlsConfigFromP12Buffer(ba []byte) (*TlsPackage, error) {
	DebugFunc()

	_, _, err := VerifyP12(ba, pkcs12.DefaultPassword)
	if Error(err) {
		return nil, err
	}

	key, cert, caCerts, err := pkcs12.DecodeChain(ba, pkcs12.DefaultPassword)
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

	caCertPool, _ := x509.SystemCertPool()
	if caCertPool == nil {
		caCertPool = x509.NewCertPool()
	}

	caCertPool.AppendCertsFromPEM(certAsPem)

	certificate, err := tls.X509KeyPair([]byte(certAsPem), []byte(keyAsPem))
	if Error(err) {
		return nil, err
	}

	tlsConfig := tls.Config{
		Rand:                     rand.Reader,
		PreferServerCipherSuites: true,
		Certificates:             []tls.Certificate{certificate},
		RootCAs:                  caCertPool,
		ClientCAs:                caCertPool,
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256,
		},
	}

	list := []*x509.Certificate{cert}
	list = append(list, caCerts...)

	certInfos, err := CertificateInfoFromX509(list)
	if Error(err) {
		return nil, err
	}

	return &TlsPackage{
		CertificateAsPem: certAsPem,
		PrivateKeyAsPem:  keyAsPem,
		Certificate:      cert,
		PrivateKey:       key,
		P12:              ba,
		CaCerts:          caCerts,
		Info:             certInfos,
		Config:           tlsConfig,
	}, nil
}

func TLSConfigFromPem(certAsPem []byte, keyAsPem []byte) (*TlsPackage, error) {
	DebugFunc("generate TLS config from given cert and key flags")

	certBytes, _ := pem.Decode(certAsPem)
	if certBytes == nil {
		return nil, fmt.Errorf("cannot find PEM block with certificate")
	}
	keyBytes, _ := pem.Decode(keyAsPem)
	if keyBytes == nil {
		return nil, fmt.Errorf("cannot find PEM block with key")
	}

	cert, err := x509.ParseCertificate(certBytes.Bytes)
	if err != nil {
		panic("failed to parse certificate: " + err.Error())
	}

	priv, err := x509.ParsePKCS1PrivateKey(keyBytes.Bytes)
	if err != nil {
		return nil, err
	}

	p12, err := pkcs12.Encode(rand.Reader, priv, cert, nil, pkcs12.DefaultPassword)
	if Error(err) {
		return nil, err
	}

	return TlsConfigFromP12Buffer(p12)
}

func createCertificateTemplate() (*x509.Certificate, error) {
	DebugFunc()

	_, hostname, err := GetHost()
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
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   hostname,
			Organization: []string{TitleVersion(true, true, true)}},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now().Add(time.Duration(24) * time.Hour * -1),
		NotAfter:              time.Now().Add(time.Duration(10) * 365 * 24 * time.Hour),
		DNSNames:              []string{hostname, "localhost"},
		BasicConstraintsValid: true,
	}

	return &tmpl, nil
}

func createCertificate(template, parent *x509.Certificate, pub interface{}, parentPriv interface{}) (cert *x509.Certificate, certPEM []byte, err error) {
	DebugFunc()

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

func CreateTlsPackage() (*TlsPackage, error) {
	DebugFunc()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if Error(err) {
		return nil, err
	}

	certTmpl, err := createCertificateTemplate()
	if Error(err) {
		return nil, err
	}

	addrs, err := GetActiveAddrs(true)
	if Error(err) {
		return nil, err
	}

	parsedIps := make([]net.IP, 0)
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if Error(err) {
			return nil, err
		}

		parsedIps = append(parsedIps, ip)
	}

	certTmpl.IsCA = true
	certTmpl.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	certTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	certTmpl.IPAddresses = parsedIps

	_, certPEM, err := createCertificate(certTmpl, certTmpl, &key.PublicKey, key)
	if Error(err) {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	return TLSConfigFromPem(certPEM, keyPEM)
}

func GetTlsPackage() (*TlsPackage, error) {
	DebugFunc()

	muTLS.Lock()
	defer muTLS.Unlock()

	var tlsPackage *TlsPackage

	if *FlagTlsP12File != "" {
		tlsPackage, _ = TLSConfigFromP12File(*FlagTlsP12File)

		if tlsPackage != nil {
			return tlsPackage, nil
		}
	}

	cfg := GetConfiguration()

	if cfg != nil {
		p12, _ := cfg.GetFlag(FlagNameTlsP12)
		if p12 != "" {
			ba, _ := base64.StdEncoding.DecodeString(p12)

			if ba != nil {
				tlsPackage, _ = TlsConfigFromP12Buffer(ba)

				if tlsPackage != nil {
					return tlsPackage, nil
				}
			}
		}
	}

	tlsPackage, err := CreateTlsPackage()
	if Error(err) {
		return nil, err
	}

	return tlsPackage, nil
}

func VerifyP12(p12 []byte, password string) (*x509.Certificate, *rsa.PrivateKey, error) {
	privateKey, cert, err := pkcs12.Decode(p12, password)
	if err != nil {
		return nil, nil, err
	}

	err = VerifyCertificate(cert)
	if Error(err) {
		return nil, nil, err
	}

	priv, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("Expected RSA private key type")
	}

	return cert, priv, nil
}

func VerifyCertificate(cert *x509.Certificate) error {
	DebugFunc()

	var err error

	if !IsCertificateSelfSigned(cert) {
		_, err = cert.Verify(x509.VerifyOptions{})
	}

	if err == nil {
		now := time.Now()

		if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
			err = fmt.Errorf("Certificate is not valid (NotBefore: %v, NotAfter: %v)", cert.NotBefore, cert.NotAfter)
		}
	}

	return err
}

func CertificateInfoFromConnection(con *tls.Conn) (string, error) {
	DebugFunc()

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
	DebugFunc()

	txt := ""
	for i, cert := range certs {
		var header string
		if i == 0 {
			header = Translate("Certificate")
		} else {
			header = fmt.Sprintf("%s #%d ", Translate("CA Certificate"), i-1)
		}

		info := fmt.Sprintf("%s %s\n", header, strings.Repeat("-", 60-len(header)))

		certInfo, err := certinfo.CertificateText(cert)
		if Error(err) {
			continue
		}

		info += fmt.Sprintf("%s\n", certInfo)

		txt += info
	}

	return txt, nil
}

func ExportRsaPrivateKeyAsPemStr(privkey *rsa.PrivateKey) string {
	privkey_bytes := x509.MarshalPKCS1PrivateKey(privkey)
	privkey_pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkey_bytes,
		},
	)
	return string(privkey_pem)
}

func ParseRsaPrivateKeyFromPemStr(privPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

func ExportRsaPublicKeyAsPemStr(pubkey *rsa.PublicKey) (string, error) {
	pubkey_bytes, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return "", err
	}
	pubkey_pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubkey_bytes,
		},
	)

	return string(pubkey_pem), nil
}

func ParseRsaPublicKeyFromPemStr(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		break // fall through
	}
	return nil, fmt.Errorf("Key type is not RSA")
}

func IsCertificateSelfSigned(cert *x509.Certificate) bool {
	return cert.Issuer.String() == cert.Subject.String()
}
