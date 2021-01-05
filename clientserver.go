package common

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/crypto/sha3"
	"hash"
	"io"
	"net"
)

type TCPServer struct {
	useTls       bool
	useTlsVerify bool
	address      string
	Hash         hash.Hash
	Conn         net.Conn
	listener     net.Listener
}

func NewTCPServer(useTls bool, useTlsVerify bool, address string) *TCPServer {
	return &TCPServer{
		useTls:       useTls,
		useTlsVerify: useTlsVerify,
		address:      address,
		Hash:         sha3.New512(),
	}
}

func (this *TCPServer) Serve() ([]byte, error) {
	if this.useTls {
		tlsPackage, err := GetTlsPackage()
		if Error(err) {
			return nil, err
		}

		if this.useTlsVerify {
			tlsPackage.Config.ClientAuth = tls.RequireAndVerifyClientCert
		}

		this.listener, err = tls.Listen("tcp", this.address, &tlsPackage.Config)
		if Error(err) {
			return nil, err
		}
	} else {
		tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%s", this.address))
		if Error(err) {
			return nil, err
		}

		this.listener, err = net.ListenTCP("tcp", tcpAddr)
		if Error(err) {
			return nil, err
		}
	}

	var err error

	if this.useTls {
		this.Conn, err = this.listener.Accept()
		Error(err)

		tcpConn, ok := this.Conn.(*net.TCPConn)
		if ok {
			Error(tcpConn.SetLinger(0))
		}

		tlsConn, ok := this.Conn.(*tls.Conn)
		if ok {
			err := tlsConn.Handshake()

			if Error(err) {
				Error(this.Conn.Close())
			}
		}
	} else {
		this.Conn, err = this.listener.Accept()
		Error(err)
	}

	tcpConn, ok := this.Conn.(*net.TCPConn)
	if ok {
		Error(tcpConn.SetLinger(0))
	}

	defer func() {
		Error(this.Conn.Close())
		if this.listener != nil {
			Error(this.listener.Close())
		}
	}()

	this.Hash.Reset()

	_, err = io.Copy(this.Hash, this.Conn)

	return this.Hash.Sum(nil), err
}

type TCPClient struct {
	useTls       bool
	useTlsVerify bool
	address      string
	Conn         net.Conn
}

func NewTCPClient(useTls bool, useTlsVerify bool, address string) *TCPClient {
	return &TCPClient{
		useTls:       useTls,
		useTlsVerify: useTlsVerify,
		address:      address,
	}
}

func (this *TCPClient) Connect() (io.ReadWriter, error) {
	if this.useTls {
		tlsPackage, err := GetTlsPackage()
		if Error(err) {
			return nil, err
		}

		hostname, _, err := net.SplitHostPort(this.address)
		if Error(err) {
			return nil, err
		}

		if hostname == "" {
			hostname = "localhost"
		}

		// set hostname for self-signed certificates
		tlsPackage.Config.ServerName = hostname
		tlsPackage.Config.InsecureSkipVerify = !this.useTlsVerify

		this.Conn, err = tls.Dial("tcp", this.address, &tlsPackage.Config)
		if Error(err) {
			return nil, err
		}

		tlsSocket, ok := this.Conn.(*tls.Conn)
		if ok {
			if !tlsSocket.ConnectionState().HandshakeComplete {
				return nil, fmt.Errorf("TLS handshake not completed")
			}
		}
	} else {
		tcpAddr, err := net.ResolveTCPAddr("tcp", this.address)
		if Error(err) {
			return nil, err
		}

		this.Conn, err = net.DialTCP("tcp", nil, tcpAddr)
		if Error(err) {
			return nil, err
		}
	}

	return this.Conn, nil
}
