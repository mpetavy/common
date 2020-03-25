package common

import (
	"crypto/tls"
	"flag"
	"fmt"
	"golang.org/x/sys/cpu"
	"sort"
	"strings"
)

var (
	FlagTlsInsecure *bool
	hasGCMAsm       bool
	cipherSuites    []*tls.CipherSuite

	topCipherSuites []uint16
	versions        []uint16
)

const (
	tlsVersion10 = "TLS 1.0"
	tlsVersion11 = "TLS 1.1"
	tlsVersion12 = "TLS 1.2"
	tlsVersion13 = "TLS 1.3"
)

func init() {
	FlagTlsInsecure = flag.Bool("tls.insecure", false, "Use insecure TLS versions and ciphersuites")

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

		Debug("Cipher #%2d: %s %s", i, FillString(priorityInfo, 70, false, " "), FillString(topInfo, 70, false, " "))
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
