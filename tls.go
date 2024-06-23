package common

// https://www.golinuxcloud.com/tutorial-pki-certificates-authority-ocsp/
// https://sockettools.com/kb/creating-certificate-using-openssl/
// http://blog.fourthbit.com/2014/12/23/traffic-analysis-of-an-ssl-slash-tls-session/
// https://www.atidur.dev/blog/dont-trust-standard-lib/

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"github.com/grantae/certinfo"
	"golang.org/x/sys/cpu"
	"math/big"
	"net"
	"os"
	"runtime"
	"software.sslmate.com/src/go-pkcs12"
	"strings"
	"time"
)

var (
	FlagTlsInsecure    *bool
	FlagTlsVerify      *bool
	FlagTlsServername  *string
	FlagTlsMinVersion  *string
	FlagTlsMaxVersion  *string
	FlagTlsCiphers     *string
	FlagTlsPassword    *string
	FlagTlsCertificate *string
	FlagTlsMutual      *string
	FlagTlsKeyLen      *int
	defaultCiphers     []*tls.CipherSuite
	defautVersions     []uint16

	// Check the cpu flags for each platform that has optimized GCM implementations.
	// Worst case, these variables will just all be false.

	hasGCMAsmAMD64 = cpu.X86.HasAES && cpu.X86.HasPCLMULQDQ
	hasGCMAsmARM64 = cpu.ARM64.HasAES && cpu.ARM64.HasPMULL
	// Keep in sync with crypto/aes/cipher_s390x.go.
	hasGCMAsmS390X = cpu.S390X.HasAES && cpu.S390X.HasAESCBC && cpu.S390X.HasAESCTR && (cpu.S390X.HasGHASH || cpu.S390X.HasAESGCM)

	hasAESGCMHardwareSupport = runtime.GOARCH == "amd64" && hasGCMAsmAMD64 ||
		runtime.GOARCH == "arm64" && hasGCMAsmARM64 ||
		runtime.GOARCH == "s390x" && hasGCMAsmS390X
)

const (
	FlagNameTlsInsecure    = "tls.insecure"
	FlagNameTlsVerify      = "tls.verify"
	FlagNameTlsServername  = "tls.servername"
	FlagNameTlsMinVersion  = "tls.minversion"
	FlagNameTlsMaxVersion  = "tls.maxversion"
	FlagNameTlsCiphers     = "tls.ciphers"
	FlagNameTlsPassword    = "tls.password"
	FlagNameTlsCertificate = "tls.certificate"
	FlagNameTlsMutual      = "tls.mutual"
	FlagNameTlsKeylen      = "tls.keylen"
)

const (
	TlsVersion10 = "TLS1.0"
	TlsVersion11 = "TLS1.1"
	TlsVersion12 = "TLS1.2"
	TlsVersion13 = "TLS1.3"
)

func init() {
	Events.AddListener(EventInit{}, func(ev Event) {
		FlagTlsInsecure = flag.Bool(FlagNameTlsInsecure, false, "Use insecure TLS versions and cipher suites")
		FlagTlsVerify = flag.Bool(FlagNameTlsVerify, false, "Verify TLS certificates and server name")
		FlagTlsServername = flag.String(FlagNameTlsServername, "", "TLS expected servername")
		FlagTlsMinVersion = flag.String(FlagNameTlsMinVersion, TlsVersion12, "TLS min version")
		FlagTlsMaxVersion = flag.String(FlagNameTlsMaxVersion, TlsVersion12, "TLS max version")
		FlagTlsCiphers = flag.String(FlagNameTlsCiphers, "", "TLS ciphers zo use")
		FlagTlsPassword = flag.String(FlagNameTlsPassword, pkcs12.DefaultPassword, "TLS PKCS12 certificates & privkey container file (P12 format)")
		FlagTlsCertificate = flag.String(FlagNameTlsCertificate, "", "Server TLS PKCS12 certificates & privkey container file or buffer")
		FlagTlsMutual = flag.String(FlagNameTlsMutual, "", "Mutual TLS PKCS12 certificates & privkey container file or buffer")
		FlagTlsKeyLen = flag.Int(FlagNameTlsKeylen, 256, "Key length")
	})

	Events.AddListener(EventFlagsSet{}, func(ev Event) {
		initTls()
	})
}

func initTls() {
	Debug("Hardware cipher implementation available: %v", hasAESGCMHardwareSupport)

	defautVersions = make([]uint16, 0)
	if *FlagTlsInsecure {
		defautVersions = append(defautVersions, tls.VersionTLS10, tls.VersionTLS11)
	}

	defautVersions = append(defautVersions, tls.VersionTLS12, tls.VersionTLS13)

	defaultCiphers = make([]*tls.CipherSuite, 0)
	defaultCiphers = append(defaultCiphers, tls.CipherSuites()...)

	if *FlagTlsInsecure {
		defaultCiphers = append(defaultCiphers, tls.InsecureCipherSuites()...)
	}

	i := 0
	for i < len(defaultCiphers) {
		supported := false
		for _, csv := range defaultCiphers[i].SupportedVersions {
			supported = IndexOf(defautVersions, csv) != -1
			if supported {
				break
			}
		}

		if supported {
			i++
		} else {
			defaultCiphers = SliceDelete(defaultCiphers, i)
		}
	}

	for i := 0; i < len(defaultCiphers); i++ {
		Debug("Cipher priority #%02d: %s", i, TlsCipherDescription(defaultCiphers[i]))
	}
}

func TlsDefaultCiphers() []*tls.CipherSuite {
	return defaultCiphers
}

func TlsCiphersIds(ciphers []*tls.CipherSuite) []uint16 {
	ids := make([]uint16, 0)

	for _, suit := range defaultCiphers {
		ids = append(ids, suit.ID)
	}

	return ids
}

func TlsCipherNames(ciphers []*tls.CipherSuite) []string {
	names := make([]string, 0)

	for _, suit := range defaultCiphers {
		names = append(names, suit.Name)
	}

	return names
}

func TlsIdToCipher(id uint16) *tls.CipherSuite {
	for _, cs := range TlsDefaultCiphers() {
		if cs.ID == id {
			return cs
		}
	}

	return nil
}

func TlsCipherDescription(cs *tls.CipherSuite) string {
	tlsVersion := make([]string, 0)
	for _, v := range cs.SupportedVersions {
		tlsVersion = append(tlsVersion, TlsIdToVersion(v))
	}

	return fmt.Sprintf("%s [%s]%s", cs.Name, Join(tlsVersion, ","), Eval(cs.Insecure, fmt.Sprintf("[%s]", Translate("Insecure")), ""))
}

func TlsDescriptionToCipher(name string) *tls.CipherSuite {
	p := strings.Index(name, " ")
	if p != -1 {
		name = name[:p]
	}

	for _, cs := range TlsDefaultCiphers() {
		if cs.Name == name {
			return cs
		}
	}

	return nil
}

func TlsCipherSelectionsToIds(s string) []uint16 {
	list := make([]uint16, 0)

	for _, name := range strings.Split(s, ";") {
		cs := TlsDescriptionToCipher(name)
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
	case TlsVersion11:
		return tls.VersionTLS11
	case TlsVersion12:
		return tls.VersionTLS12
	case TlsVersion13:
		return tls.VersionTLS13
	}
}

func TlsIdToVersion(id uint16) string {
	switch id {
	default:
		return TlsVersion10
	case tls.VersionTLS11:
		return TlsVersion11
	case tls.VersionTLS12:
		return TlsVersion12
	case tls.VersionTLS13:
		return TlsVersion13
	}
}

func TlsVersions() []string {
	list := make([]string, 0)
	for i := range defautVersions {
		list = append(list, TlsIdToVersion(defautVersions[i]))
	}

	return list
}

func TlsDebugConnection(typ string, tlsConn *tls.Conn) {
	if *FlagLogIO {
		connstate := tlsConn.ConnectionState()

		Debug("TLS connection %s: Version : %s\n", typ, TlsIdToVersion(connstate.Version))
		Debug("TLS connection %s: CipherSuite : %v\n", typ, TlsCipherDescription(TlsIdToCipher(connstate.CipherSuite)))
		Debug("TLS connection %s: HandshakeComplete : %v\n", typ, connstate.HandshakeComplete)
		Debug("TLS connection %s: DidResume : %v\n", typ, connstate.DidResume)
		Debug("TLS connection %s: NegotiatedProtocol : %x\n", typ, connstate.NegotiatedProtocol)
		Debug("TLS connection %s: ServerName : %s\n", typ, connstate.ServerName)

		for i, peercert := range connstate.PeerCertificates {
			Debug("TLS connection info %s: PeerCertificate %d : %+v\n", typ, i, *peercert)
		}

		for i := range connstate.VerifiedChains {
			for k, rootcert := range connstate.VerifiedChains[i] {
				Debug("TLS connection info %s: Verified Chains %d : %+v\n", typ, k, rootcert)
			}
		}
	}
}

func PrivateKeyAsPEM(privateKey *ecdsa.PrivateKey) ([]byte, error) {
	ba, err := x509.MarshalECPrivateKey(privateKey)
	if Error(err) {
		return nil, err
	}

	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: ba,
		},
	), nil
}

func CertificateAsPEM(tlsCertificate *tls.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tlsCertificate.Certificate[0]})
}

func X509toTlsCertificate(certificate *x509.Certificate, privateKey *ecdsa.PrivateKey) (*tls.Certificate, error) {
	keyPEM, err := PrivateKeyAsPEM(privateKey)
	if Error(err) {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})

	certTls, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if Error(err) {
		return nil, err
	}

	return &certTls, nil
}

func TlsToX509Certificate(certificate []byte) (*x509.Certificate, error) {
	return x509.ParseCertificate(certificate)
}

func TlsConfigFromP12(ba []byte, password string) (*tls.Config, error) {
	// the priority on entities inside p12 must be honored
	// 1st	  private key
	// 2nd	  computer certificate
	// 3d..n  CA certificates (will be ignored by app)

	DebugFunc()

	p12PrivateKey, p12Cert, p12RootCerts, err := VerifyP12(ba, password)
	if Error(err) {
		return nil, err
	}

	var rootCertPool *x509.CertPool

	if len(p12RootCerts) > 0 {
		rootCertPool, _ = x509.SystemCertPool()
		if rootCertPool == nil {
			rootCertPool = x509.NewCertPool()
		}

		for _, rootCert := range p12RootCerts {
			rootCertPool.AddCert(rootCert)
		}
	}

	certificate, err := X509toTlsCertificate(p12Cert, p12PrivateKey)
	if Error(err) {
		return nil, err
	}

	return &tls.Config{
		Rand:         rand.Reader,
		Certificates: []tls.Certificate{*certificate},
		RootCAs:      rootCertPool,
		MinVersion:   TlsVersionToId(*FlagTlsMinVersion),
		MaxVersion:   TlsVersionToId(*FlagTlsMaxVersion),
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256,
		},
	}, nil
}

func TlsConfigToP12(tlsConfig *tls.Config, password string) ([]byte, error) {
	certPEM := CertificateAsPEM(&tlsConfig.Certificates[0])
	certBytes, _ := pem.Decode(certPEM)
	if certBytes == nil {
		return nil, fmt.Errorf("cannot find PEM block with certificate")
	}

	keyPEM, err := PrivateKeyAsPEM(tlsConfig.Certificates[0].PrivateKey.(*ecdsa.PrivateKey))
	if Error(err) {
		return nil, err
	}
	keyBytes, _ := pem.Decode(keyPEM)
	if keyBytes == nil {
		return nil, fmt.Errorf("cannot find PEM block with key")
	}

	cert, err := x509.ParseCertificate(certBytes.Bytes)
	if Error(err) {
		return nil, err
	}

	priv, err := x509.ParseECPrivateKey(keyBytes.Bytes)
	if Error(err) {
		return nil, err
	}

	p12, err := pkcs12.Encode(rand.Reader, priv, cert, nil, password)
	if Error(err) {
		return nil, err
	}

	return p12, nil
}

func TlsConfigFromPEM(certPEM []byte, keyPEM []byte, password string) (*tls.Config, error) {
	DebugFunc("generate TLS config from given cert and key flags")

	certBytes, _ := pem.Decode(certPEM)
	if certBytes == nil {
		return nil, fmt.Errorf("cannot find PEM block with certificate")
	}
	keyBytes, _ := pem.Decode(keyPEM)
	if keyBytes == nil {
		return nil, fmt.Errorf("cannot find PEM block with key")
	}

	cert, err := x509.ParseCertificate(certBytes.Bytes)
	if Error(err) {
		return nil, err
	}

	priv, err := x509.ParseECPrivateKey(keyBytes.Bytes)
	if Error(err) {
		return nil, err
	}

	p12, err := pkcs12.Encode(rand.Reader, priv, cert, nil, password)
	if Error(err) {
		return nil, err
	}

	return TlsConfigFromP12(p12, password)
}

func CreateTlsConfig(keylen int, password string) (*tls.Config, error) {
	DebugFunc()

	var curve elliptic.Curve

	switch keylen {
	case 224:
		curve = elliptic.P224()
	case 256:
		curve = elliptic.P256()
	case 384:
		curve = elliptic.P384()
	case 521:
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unknown key len: %d", keylen)
	}

	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if Error(err) {
		return nil, err
	}

	hostname, _, hostInfos, err := GetHostInfos()
	if Error(err) {
		return nil, err
	}

	// generate a random serial number (a real cert authority would have some logic behind this)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if Error(err) {
		return nil, err
	}

	certTmpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   hostname,
			Organization: []string{TitleVersion(true, true, true)}},
		NotBefore:             CalcDeadline(time.Now(), time.Duration(24)*time.Hour*-1),
		NotAfter:              CalcDeadline(time.Now(), time.Duration(10)*365*24*time.Hour),
		DNSNames:              []string{hostname, "localhost"},
		BasicConstraintsValid: true,
	}

	ips := make([]net.IP, 0)
	for _, hostInfo := range hostInfos {
		ips = append(ips, hostInfo.IPNet.IP)
	}

	certTmpl.IsCA = false
	certTmpl.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	certTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	certTmpl.IPAddresses = ips

	certDER, err := x509.CreateCertificate(rand.Reader, certTmpl, certTmpl, &key.PublicKey, key)
	if Error(err) {
		return nil, err
	}

	certX509, err := x509.ParseCertificate(certDER)
	if Error(err) {
		return nil, err
	}

	certTls, err := X509toTlsCertificate(certX509, key)
	if Error(err) {
		return nil, err
	}

	certPEM := CertificateAsPEM(certTls)
	keyPEM, err := PrivateKeyAsPEM(certTls.PrivateKey.(*ecdsa.PrivateKey))
	if Error(err) {
		return nil, err
	}

	return TlsConfigFromPEM(certPEM, keyPEM, password)
}

func NewTlsConfigFromFlags() (*tls.Config, error) {
	tlsConfigFromFlags, err := NewTlsConfig(
		*FlagTlsVerify,
		*FlagTlsServername,
		*FlagTlsMinVersion,
		*FlagTlsMaxVersion,
		*FlagTlsCiphers,
		*FlagTlsPassword,
		*FlagTlsCertificate,
		*FlagTlsMutual,
		*FlagTlsKeyLen)
	if Error(err) {
		return nil, err
	}

	Debug("%+v", tlsConfigFromFlags)

	return tlsConfigFromFlags, nil
}

func readP12FromFileOrBuffer(fileOrBuffer string, password string) (*tls.Config, error) {
	if FileExists_(fileOrBuffer) {
		ba, err := os.ReadFile(fileOrBuffer)
		if Error(err) {
			return nil, err
		}

		tlsConfig, err := TlsConfigFromP12(ba, password)
		if Error(err) {
			return nil, err
		}

		return tlsConfig, nil
	}

	tlsConfig, err := TlsConfigFromP12([]byte(fileOrBuffer), password)
	if Error(err) {
		return nil, err
	}

	return tlsConfig, nil
}

func NewTlsConfig(
	certificateVerify bool,
	serverName string,
	minVersion string,
	maxVersion string,
	ciphers string,
	password string,
	certificate string,
	mutual string,
	keylen int) (*tls.Config, error) {
	DebugFunc()

	var tlsConfig *tls.Config
	var err error

	if certificate != "" {
		tlsConfig, err = readP12FromFileOrBuffer(certificate, password)
		if Error(err) {
			return nil, err
		}
	} else {
		tlsConfig, err = CreateTlsConfig(keylen, password)
		if Error(err) {
			return nil, err
		}
	}

	tlsConfig.InsecureSkipVerify = !certificateVerify
	tlsConfig.ServerName = serverName
	tlsConfig.MinVersion = TlsVersionToId(minVersion)
	tlsConfig.MaxVersion = TlsVersionToId(maxVersion)
	if ciphers != "" {
		tlsConfig.CipherSuites = TlsCipherSelectionsToIds(ciphers)
	} else {
		tlsConfig.CipherSuites = TlsCiphersIds(TlsDefaultCiphers())
	}

	if mutual != "" {
		packageMutual, err := readP12FromFileOrBuffer(mutual, password)
		if Error(err) {
			return nil, err
		}

		tlsConfig.ClientCAs = packageMutual.RootCAs
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	txt, err := TlsCertificateInfos(tlsConfig)
	if Error(err) {
		return nil, err
	}

	DebugFunc(txt)

	return tlsConfig, nil
}

func VerifyP12(ba []byte, password string) (privateKey *ecdsa.PrivateKey, certificate *x509.Certificate, caCerts []*x509.Certificate, err error) {
	p12PrivateKey, p12Cert, p12RootCerts, err := pkcs12.DecodeChain(ba, password)
	if Error(err) {
		return nil, nil, nil, err
	}

	if len(p12Cert.Subject.CommonName) == 0 {
		return nil, nil, nil, fmt.Errorf(Translate("Certificate does not contain a x509 CommonName attribute"))
	}

	if len(p12Cert.DNSNames) == 0 {
		return nil, nil, nil, fmt.Errorf(Translate("Certificate does not contain a x509 Subject Alternate Name (SAN) DNSName attribute"))
	}

	if !IsCertificateSelfSigned(p12Cert) {
		var rootCertPool *x509.CertPool

		if len(p12RootCerts) > 0 {
			rootCertPool, _ = x509.SystemCertPool()
			if rootCertPool == nil {
				rootCertPool = x509.NewCertPool()
			}

			for _, rootCert := range p12RootCerts {
				rootCertPool.AddCert(rootCert)
			}
		}

		_, err = p12Cert.Verify(x509.VerifyOptions{
			DNSName:                   "",
			Intermediates:             nil,
			Roots:                     rootCertPool,
			CurrentTime:               time.Time{},
			KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			MaxConstraintComparisions: 0,
		})
		if Error(err) {
			return nil, nil, nil, err
		}
	}

	now := time.Now()
	if now.Before(p12Cert.NotBefore) || now.After(p12Cert.NotAfter) {
		return nil, nil, nil, fmt.Errorf("Certificate is not valid (NotBefore: %v, NotAfter: %v)", p12Cert.NotBefore, p12Cert.NotAfter)
	}

	p12PrivateKeyEcdsa, ok := p12PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, nil, nil, fmt.Errorf("Unexpected private key type: %T", p12PrivateKey)
	}

	return p12PrivateKeyEcdsa, p12Cert, p12RootCerts, nil
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

func TlsCertificateInfos(tlsConfig *tls.Config) (string, error) {
	DebugFunc()

	var txt strings.Builder

	for i, cert := range tlsConfig.Certificates {
		var header string
		if i == 0 {
			header = Translate("Certificate")
		} else {
			header = fmt.Sprintf("%s #%d ", Translate("CA Certificate"), i-1)
		}

		info := fmt.Sprintf("%s %s\n", header, strings.Repeat("-", 60-len(header)))

		cert, err := TlsToX509Certificate(cert.Certificate[0])
		if Error(err) {
			return "", err
		}

		certInfo, err := certinfo.CertificateText(cert)
		if Error(err) {
			continue
		}

		info += fmt.Sprintf("%s\n", certInfo)

		txt.WriteString(info)
	}

	return txt.String(), nil
}

func IsCertificateSelfSigned(cert *x509.Certificate) bool {
	b := cert.Issuer.String() == cert.Subject.String()

	DebugFunc(b)

	return b
}
