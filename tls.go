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
	"golang.org/x/sys/cpu"
	"io/ioutil"
	"math/big"
	"net"
	"software.sslmate.com/src/go-pkcs12"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	FlagTlsInsecure *bool
	hasGCMAsm       bool
	cipherSuites    []*tls.CipherSuite

	topCipherSuites []uint16
	versions        []uint16
)

const (
	FlagNameTlsInsecure = "tls.insecure"
)

const (
	tlsVersion10 = "TLS 1.0"
	tlsVersion11 = "TLS 1.1"
	tlsVersion12 = "TLS 1.2"
	tlsVersion13 = "TLS 1.3"
)

func init() {
	FlagTlsInsecure = flag.Bool(FlagNameTlsInsecure, false, "Use insecure TLS versions and ciphersuites")

	Events.NewFuncReceiver(EventFlagsSet{}, func(ev Event) {
		initTls()
	})

	Events.NewFuncReceiver(EventAppRestart{}, func(ev Event) {
		initTls()
	})
}

func initDefaultCipherSuites() {
	// Check the cpu flags for each platform that has optimized GCM implementations.
	// Worst case, these variables will just all be false.

	var (
		hasGCMAsmAMD64 = cpu.X86.HasAES && cpu.X86.HasPCLMULQDQ
		hasGCMAsmARM64 = cpu.ARM64.HasAES && cpu.ARM64.HasPMULL
		// Keep in sync with crypto/aes/cipher_s390x.go.
		hasGCMAsmS390X = cpu.S390X.HasAES && cpu.S390X.HasAESCBC && cpu.S390X.HasAESCTR && (cpu.S390X.HasGHASH || cpu.S390X.HasAESGCM)

		hasGCMAsm = hasGCMAsmAMD64 || hasGCMAsmARM64 || hasGCMAsmS390X
	)

	if hasGCMAsm {
		// If AES-GCM hardware is provided then prioritise AES-GCM
		// cipher suites.
		topCipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,

			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
		}
	} else {
		// Without AES-GCM hardware, we put the ChaCha20-Poly1305
		// cipher suites first.
		topCipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,

			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
		}
	}
}

func initTls() {
	initDefaultCipherSuites()

	cipherSuites = make([]*tls.CipherSuite, 0)
	versions = make([]uint16, 0)

	cipherSuites = append(cipherSuites, tls.CipherSuites()...)

	if *FlagTlsInsecure {
		cipherSuites = append(cipherSuites, tls.InsecureCipherSuites()...)

		versions = append(versions, tls.VersionTLS10, tls.VersionTLS11)
	}

	versions = append(versions, tls.VersionTLS12, tls.VersionTLS13)

	i := 0
	for i < len(cipherSuites) {
		supported := false
		for _, csv := range cipherSuites[i].SupportedVersions {
			supported = IndexOf(versions, csv) != -1
			if supported {
				break
			}
		}

		if supported {
			i++
		} else {
			cipherSuites = append(cipherSuites[:i], cipherSuites[i+1:]...)
		}
	}

	sort.SliceStable(cipherSuites, func(i, j int) bool {
		oi := orderOfCipherSuite(cipherSuites[i].ID)
		oj := orderOfCipherSuite(cipherSuites[j].ID)

		switch {
		case oi != -1 && oj != -1:
			return oi < oj
		case oi == -1 && oj == -1:
			return strings.Compare(cipherSuites[i].Name, cipherSuites[j].Name) == -1
		case oi != -1:
			return true
		default:
			return false
		}
	})

	Debug("Cipher hasGCMAsm: %v", hasGCMAsm)

	max := Max(len(topCipherSuites), len(cipherSuites))
	for i := 0; i < max; i++ {
		topInfo := ""
		priorityInfo := ""

		if i < len(topCipherSuites) {
			topInfo = TlsCipherSuiteToInfo(TlsIdToCipherSuite(topCipherSuites[i]))
		}

		if i < len(cipherSuites) {
			priorityInfo = TlsCipherSuiteToInfo(cipherSuites[i])
		}

		Debug("Cipher #%02d: %s %s", i, FillString(priorityInfo, 70, false, " "), FillString(topInfo, 70, false, " "))
	}
}

func TlsCipherSuites() []*tls.CipherSuite {
	return cipherSuites
}

func TlsIdToCipherSuite(id uint16) *tls.CipherSuite {
	for _, cs := range TlsCipherSuites() {
		if cs.ID == id {
			return cs
		}
	}

	return nil
}

func TlsCipherSuiteToInfo(cs *tls.CipherSuite) string {
	tlsVersion := make([]string, 0)
	for _, v := range cs.SupportedVersions {
		tlsVersion = append(tlsVersion, TlsIdToVersion(v))
	}

	return fmt.Sprintf("%s [%s]%s", cs.Name, Join(tlsVersion, ","), Eval(cs.Insecure, fmt.Sprintf("[%s]", Translate("Insecure")), "").(string))
}

func TlsInfoToCipherSuite(name string) *tls.CipherSuite {
	p := strings.Index(name, " ")
	if p != -1 {
		name = name[:p]
	}

	for _, cs := range TlsCipherSuites() {
		if cs.Name == name {
			return cs
		}
	}

	return nil
}

func orderOfCipherSuite(id uint16) int {
	for i, cs := range topCipherSuites {
		if cs == id {
			return i
		}
	}

	return -1
}

func TlsInfosToCipherSuites(s string) []uint16 {
	list := make([]uint16, 0)

	for _, name := range strings.Split(s, ";") {
		cs := TlsInfoToCipherSuite(name)
		if cs != nil {
			list = append(list, cs.ID)
		}
	}

	return list
}

func TlsVersionToId(s string) uint16 {
	switch s {
	default:
		return tls.VersionTLS10
	case tlsVersion11:
		return tls.VersionTLS11
	case tlsVersion12:
		return tls.VersionTLS12
	case tlsVersion13:
		return tls.VersionTLS13
	}
}

func TlsIdToVersion(id uint16) string {
	switch id {
	default:
		return tlsVersion10
	case tls.VersionTLS11:
		return tlsVersion11
	case tls.VersionTLS12:
		return tlsVersion12
	case tls.VersionTLS13:
		return tlsVersion13
	}
}

func TlsVersions() []string {
	list := make([]string, 0)
	for i := range versions {
		list = append(list, TlsIdToVersion(versions[i]))
	}

	return list
}

func DebugTlsConnectionInfo(typ string, tlsConn *tls.Conn) {
	connstate := tlsConn.ConnectionState()

	Debug("TLS connection info %s: Version : %s\n", typ, TlsIdToVersion(connstate.Version))
	Debug("TLS connection info %s: CipherSuite : %v\n", typ, TlsCipherSuiteToInfo(TlsIdToCipherSuite(connstate.CipherSuite)))
	Debug("TLS connection info %s: HandshakeComplete : %v\n", typ, connstate.HandshakeComplete)
	Debug("TLS connection info %s: DidResume : %v\n", typ, connstate.DidResume)
	Debug("TLS connection info %s: NegotiatedProtocol : %x\n", typ, connstate.NegotiatedProtocol)
	Debug("TLS connection info %s: NegotiatedProtocolIsMutual : %v\n", typ, connstate.NegotiatedProtocolIsMutual)
	Debug("TLS connection info %s: ServerName : %s\n", typ, connstate.ServerName)

	for i := range connstate.PeerCertificates {
		peercert := &connstate.PeerCertificates[i]
		Debug("TLS connection info %s: PeerCertificate %d : %d\n", typ, i, peercert)
	}

	for r := range connstate.VerifiedChains {
		vchains := &connstate.VerifiedChains[r]
		Debug("TLS connection info %s: Verified Chains %d : %d\n", typ, r, vchains)
	}
}

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

	addrs, err := GetHostAddrs(true, nil)
	if Error(err) {
		return nil, err
	}

	parsedIps := make([]net.IP, 0)
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.Addr.String())
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

	if *FlagTlsP12 != "" {
		ba, _ := base64.StdEncoding.DecodeString(*FlagTlsP12)

		if ba != nil {
			tlsPackage, _ = TlsConfigFromP12Buffer(ba)

			if tlsPackage != nil {
				return tlsPackage, nil
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
