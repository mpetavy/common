package common

import (
	"context"
	"crypto/tls"
	"fmt"
	"go.bug.st/serial"
	"golang.org/x/crypto/sha3"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Endpoint interface {
	Start() error
	Stop() error
}

type EndpointConnector func() (io.ReadWriteCloser, error)

func IsTTYDevice(device string) bool {
	return len(device) > 0 && (strings.Contains(device, ",") || !strings.Contains(device, ":"))
}

func NewEndpoint(device string, isClient bool, tlsConfig *tls.Config) (Endpoint, EndpointConnector, error) {
	var ep Endpoint
	var connector EndpointConnector

	if IsTTYDevice(device) {
		tty, err := NewTTY(device)
		if Error(err) {
			return nil, nil, err
		}

		ep = tty

		connector = func() (io.ReadWriteCloser, error) {
			return tty.Connect()
		}

		return ep, connector, nil
	} else {
		if isClient {
			networkClient, err := NewNetworkClient(device, tlsConfig)
			if Error(err) {
				return nil, nil, err
			}

			connector = func() (io.ReadWriteCloser, error) {
				return networkClient.Connect()
			}

			ep = networkClient
		} else {
			networkServer, err := NewNetworkServer(device, tlsConfig)
			if Error(err) {
				return nil, nil, err
			}

			connector = func() (io.ReadWriteCloser, error) {
				return networkServer.Connect()
			}

			ep = networkServer
		}

		return ep, connector, nil
	}
}

type NetworkConnection struct {
	Socket net.Conn
}

func (networkConnection *NetworkConnection) Read(p []byte) (n int, err error) {
	return networkConnection.Socket.Read(p)
}

func (networkConnection *NetworkConnection) Write(p []byte) (n int, err error) {
	return networkConnection.Socket.Write(p)
}

func (networkConnection *NetworkConnection) Close() error {
	defer func() {
		networkConnection.Socket = nil
	}()

	if networkConnection.Socket != nil {
		err := networkConnection.Socket.Close()
		if Error(err) {
			return err
		}
	}

	return nil
}

type NetworkClient struct {
	address   string
	tlsConfig *tls.Config
}

func NewNetworkClient(address string, tlsConfig *tls.Config) (*NetworkClient, error) {
	networkClient := &NetworkClient{
		address:   address,
		tlsConfig: tlsConfig,
	}

	return networkClient, nil
}

func (networkClient *NetworkClient) Start() error {
	return nil
}

func (networkClient *NetworkClient) Stop() error {
	return nil
}

func (networkClient *NetworkClient) Connect() (*NetworkConnection, error) {
	if networkClient.tlsConfig != nil {
		Debug("Dial TLS connection: %s...", networkClient.address)

		socket, err := tls.DialWithDialer(&net.Dialer{Deadline: time.Now().Add(MillisecondToDuration(*FlagIoConnectTimeout))}, "tcp", networkClient.address, networkClient.tlsConfig)
		if Error(err) {
			return nil, err
		}

		return &NetworkConnection{
			Socket: socket,
		}, nil
	} else {
		Debug("Dial connection: %s...", networkClient.address)

		socket, err := net.DialTimeout("tcp", networkClient.address, MillisecondToDuration(*FlagIoConnectTimeout))
		if Error(err) {
			return nil, err
		}

		return &NetworkConnection{
			Socket: socket,
		}, nil
	}
}

type NetworkServer struct {
	mu        sync.Mutex
	address   string
	tlsConfig *tls.Config
	listener  net.Listener
}

func NewNetworkServer(address string, tlsConfig *tls.Config) (*NetworkServer, error) {
	networkServer := &NetworkServer{
		mu:        sync.Mutex{},
		address:   address,
		tlsConfig: tlsConfig,
		listener:  nil,
	}

	return networkServer, nil
}

func (networkServer *NetworkServer) Start() error {
	networkServer.mu.Lock()
	defer networkServer.mu.Unlock()

	ips, err := GetHostInfos(true, false, nil)
	if Error(err) {
		return err
	}

	Debug("Local IPs: %v", ips)

	if networkServer.tlsConfig != nil {
		Debug("Create TLS listener: %s...", networkServer.address)

		networkServer.listener, err = tls.Listen("tcp", networkServer.address, networkServer.tlsConfig)
		if Error(err) {
			return err
		}
	} else {
		tcpAddr, err := net.ResolveTCPAddr("tcp", networkServer.address)
		if Error(err) {
			return err
		}

		Debug("Create TLS listener: %s ...", networkServer.address)

		networkServer.listener, err = net.ListenTCP("tcp", tcpAddr)
		if Error(err) {
			return err
		}
	}

	return nil
}

func (networkServer *NetworkServer) Stop() error {
	networkServer.mu.Lock()
	defer networkServer.mu.Unlock()

	defer func() {
		networkServer.listener = nil
	}()

	if networkServer.listener != nil {
		err := networkServer.listener.Close()
		if Error(err) {
			return err
		}
	}

	return nil
}

func (networkServer *NetworkServer) Connect() (*NetworkConnection, error) {
	Debug("Accept connection ...")

	socket, err := networkServer.listener.Accept()
	if IsErrNetClosing(err) || DebugError(err) {
		return nil, err
	}

	Debug("Connected: %s", socket.RemoteAddr().String())

	return &NetworkConnection{
		Socket: socket,
	}, nil
}

func (this *NetworkServer) Serve() ([]byte, error) {
	err := this.Start()
	if Error(err) {
		return nil, err
	}

	defer func() {
		Error(this.Stop())
	}()

	w, err := this.Connect()
	if Error(err) {
		return nil, err
	}

	defer func() {
		Error(w.Close())
	}()

	hash := sha3.New512()

	_, err = io.Copy(hash, w)

	return hash.Sum(nil), err
}

type TTYConnection struct {
	port serial.Port
}

func (ttyConnection *TTYConnection) Read(p []byte) (n int, err error) {
	return ttyConnection.port.Read(p)
}

func (ttyConnection *TTYConnection) Write(p []byte) (n int, err error) {
	return ttyConnection.port.Write(p)
}

func (ttyConnection *TTYConnection) Close() error {
	defer func() {
		ttyConnection.port = nil
	}()

	if ttyConnection.port != nil {
		err := ttyConnection.port.Close()
		if Error(err) {
			return err
		}

		time.Sleep(time.Millisecond * 200)
	}

	return nil
}

type TTY struct {
	device string
}

func NewTTY(device string) (*TTY, error) {
	tty := &TTY{
		device: device,
	}

	tty.device = device

	return tty, nil
}

func (tty *TTY) Start() error {
	return nil
}

func (tty *TTY) Stop() error {
	return nil
}

func (tty *TTY) Connect() (io.ReadWriteCloser, error) {
	Debug("Connected: %s", tty.device)

	serialPort, mode, err := evaluateTTYOptions(tty.device)
	if Error(err) {
		return nil, err
	}

	port, err := serial.Open(serialPort, mode)
	if Error(err) {
		return nil, err
	}

	err = port.ResetInputBuffer()
	if Error(err) {
		return nil, err
	}

	err = port.ResetOutputBuffer()
	if Error(err) {
		return nil, err
	}

	return &TTYConnection{
		port: port,
	}, nil
}

func evaluateTTYOptions(device string) (string, *serial.Mode, error) {
	ss := strings.Split(device, ",")

	baudrate := 9600
	databits := 8
	stopbits := serial.OneStopBit
	paritymode := serial.NoParity
	pm := "N"
	sb := "1"

	var portname string
	var err error

	portname = ss[0]
	if len(ss) > 1 {
		baudrate, err = strconv.Atoi(ss[1])
		if err != nil || IndexOf([]string{"50", "75", "110", "134", "150", "200", "300", "600", "1200", "1800", "2400", "4800", "7200", "9600", "14400", "19200", "28800", "38400", "57600", "76800", "115200"}, ss[1]) == -1 {
			err = fmt.Errorf("invalid baud rate: %s", ss[1])
		}

		if Error(nil) {
			return "", nil, err
		}
	}
	if len(ss) > 2 {
		databits, err = strconv.Atoi(ss[2])
		if err != nil {
			err = fmt.Errorf("invalid databits: %s", ss[2])

			if Error(nil) {
				return "", nil, err
			}
		}
	}
	if len(ss) > 3 {
		pm = strings.ToUpper(ss[3][:1])

		switch pm {
		case "N":
			paritymode = serial.NoParity
		case "O":
			paritymode = serial.OddParity
		case "E":
			paritymode = serial.EvenParity
		default:
			err = fmt.Errorf("invalid partitymode: %s", pm)

			if Error(nil) {
				return "", nil, err
			}
		}
	}

	if len(ss) > 4 {
		sb = strings.ToUpper(ss[4][:1])

		switch sb {
		case "1":
			stopbits = serial.OneStopBit
		case "1.5":
			stopbits = serial.OnePointFiveStopBits
		case "2":
			stopbits = serial.TwoStopBits
		default:
			return "", nil, fmt.Errorf("invalid stopbits: %s", sb)
		}
	}

	Debug("Use serial port %s: %d %d %s %s", portname, baudrate, databits, pm, sb)

	return portname, &serial.Mode{
		BaudRate: baudrate,
		DataBits: databits,
		Parity:   paritymode,
		StopBits: stopbits,
	}, nil
}

func DataTransfer(leftName string, left io.ReadWriter, rightName string, right io.ReadWriter) {
	DebugFunc("start")

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine())

		_, err := CopyBuffer(cancel, fmt.Sprintf("%s <- %s", leftName, rightName), left, right, 0)

		DebugError(err)
	}()

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine())

		_, err := CopyBuffer(cancel, fmt.Sprintf("%s -> %s", leftName, rightName), right, left, 0)

		DebugError(err)
	}()

	<-ctx.Done()

	DebugFunc("stop")
}
