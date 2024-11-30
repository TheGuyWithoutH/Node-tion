package udp

import (
	"Node-tion/backend/transport"
	"golang.org/x/xerrors"
	"net"
	"sync"
	"time"
)

// It is advised to define a constant (max) size for all relevant byte buffers, e.g:
const bufSize = 65000

// NewUDP returns a new udp transport implementation.
func NewUDP() transport.Transport {
	return &UDP{}
}

// UDP implements a transport layer using UDP
//
// - implements transport.Transport
type UDP struct{}

// CreateSocket implements transport.Transport
func (n *UDP) CreateSocket(address string) (transport.ClosableSocket, error) {
	updAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err // failed to resolve address
	}
	conn, err := net.ListenUDP("udp", updAddr)
	if err != nil {
		return nil, err // failed to listen on address
	}

	return &Socket{
		conn: conn,
		addr: conn.LocalAddr().String(),
		ins:  make([]transport.Packet, 0),
		outs: make([]transport.Packet, 0),
		mu:   sync.Mutex{},
	}, nil
}

// Socket implements a network socket using UDP.
//
// - implements transport.Socket
// - implements transport.ClosableSocket
type Socket struct {
	conn *net.UDPConn
	addr string
	ins  []transport.Packet
	outs []transport.Packet
	mu   sync.Mutex
}

// Close implements transport.Socket. It returns an error if already closed.
func (s *Socket) Close() error {
	// close the connection
	err := s.conn.Close()
	if err != nil {
		return err // failed to close connection
	}
	return nil
}

// Send implements transport.Socket
func (s *Socket) Send(dest string, pkt transport.Packet, timeout time.Duration) error {
	udpAddr, err := net.ResolveUDPAddr("udp", dest)
	if err != nil {
		return xerrors.Errorf("failed to resolve address: %w", err)
	}

	// timeout
	if timeout > 0 {
		err = s.conn.SetWriteDeadline(time.Now().Add(timeout))
		if err != nil {
			// write error msg formatted with the error
			return xerrors.Errorf("failed to set write deadline: %w", err)

		}
	}

	pktBytes, err := pkt.Marshal()
	if err != nil {
		return xerrors.Errorf("failed to marshal packet: %w", err)
	}
	// send the packet
	_, err = s.conn.WriteToUDP(pktBytes, udpAddr)
	if err != nil {
		return xerrors.Errorf("failed to write packet: %w", err)
	}

	// add the packet to the outs
	s.mu.Lock()
	s.outs = append(s.outs, pkt.Copy()) // make a deep copy of the packet
	s.mu.Unlock()

	return nil
}

// Recv implements transport.Socket. It blocks until a packet is received, or
// the timeout is reached. In the case the timeout is reached, return a
// TimeoutErr.
func (s *Socket) Recv(timeout time.Duration) (transport.Packet, error) {
	buf := make([]byte, bufSize)

	if timeout > 0 {
		err := s.conn.SetReadDeadline(time.Now().Add(timeout))
		if err != nil {
			return transport.Packet{}, transport.TimeoutError(timeout)
		}
	}

	// read the packet
	n, _, err := s.conn.ReadFromUDP(buf)
	if err != nil {
		return transport.Packet{}, transport.TimeoutError(timeout)
	}

	pkt := transport.Packet{}
	// unmarshal the packet
	err = pkt.Unmarshal(buf[:n])
	if err != nil {
		return transport.Packet{}, transport.TimeoutError(timeout)
	}

	// add the packet to the ins
	s.mu.Lock()
	s.ins = append(s.ins, pkt.Copy()) // make a deep copy of the packet
	s.mu.Unlock()

	return pkt, nil
}

// GetAddress implements transport.Socket. It returns the address assigned. Can
// be useful in the case one provided a :0 address, which makes the system use a
// random free port.
func (s *Socket) GetAddress() string {
	// no need to lock since the address is immutable
	return s.addr
}

// GetIns implements transport.Socket
func (s *Socket) GetIns() []transport.Packet {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ins
}

// GetOuts implements transport.Socket
func (s *Socket) GetOuts() []transport.Packet {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outs
}
