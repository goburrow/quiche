package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/goburrow/quiche"
)

const maxTokenLen = 64

func listenUDP(addr string) (net.PacketConn, error) {
	localAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp", localAddr)
}

func listen(config *quiche.Config, listenAddr, root string) error {
	socket, err := listenUDP(listenAddr)
	if err != nil {
		return err
	}
	defer socket.Close()
	s := server{
		config: config,
		socket: socket,
		conns:  make(map[string]serverConn),
	}
	log.Printf("listening: %v", socket.LocalAddr())
	return s.listen()
}

type serverConn struct {
	addr net.Addr
	conn *quiche.Connection
}

type server struct {
	config *quiche.Config
	socket net.PacketConn
	conns  map[string]serverConn

	noRetry bool
}

func (s *server) listen() error {
	buf := make([]byte, bufferSize)
	header := quiche.Header{
		SCID:  make([]byte, quiche.MaxConnIDLen),
		DCID:  make([]byte, quiche.MaxConnIDLen),
		Token: make([]byte, maxTokenLen),
	}
	for {
		deadline := s.readDeadline()
		err := s.socket.SetReadDeadline(deadline)
		if err != nil {
			return err
		}
		n, addr, err := s.socket.ReadFrom(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				for _, c := range s.conns {
					c.conn.OnTimeout()
				}
			} else {
				return err
			}
		} else {
			log.Printf("got %d bytes", n)
			s.recv(buf[:n], addr, &header)
		}
		s.send(buf[:maxDatagramSize])
		s.close()
	}
}

func (s *server) readDeadline() time.Time {
	var minTimeout time.Duration
	for _, c := range s.conns {
		timeout := c.conn.Timeout()
		if timeout < minTimeout {
			minTimeout = timeout
		}
	}
	if minTimeout > 0 {
		return time.Now().Add(minTimeout)
	}
	return time.Time{}
}

func (s *server) recv(buf []byte, addr net.Addr, h *quiche.Header) {
	err := s.headerInfo(buf, h)
	if err != nil {
		log.Printf("%s failed to parse header: %v", addr, err)
		return
	}
	log.Printf("%s packet=%x scid=%x dcid=%x", addr,
		h.Type, h.SCID, h.DCID)
	c, ok := s.conns[string(h.DCID)]
	if !ok {
		if h.Version != quiche.ProtocolVersion {
			err = s.negotiate(addr, h, buf)
			if err != nil {
				log.Printf("%s failed to write version negotiation: %v", addr, err)
			} else {
				log.Printf("%s negotiate version: %x", addr, h.Version)
			}
			return
		}
		var scid, odcid []byte
		if s.noRetry {
			scid = newConnID()
		} else {
			if len(h.Token) == 0 {
				scid = newConnID()
				err = s.retry(addr, h, scid, buf)
				if err != nil {
					log.Printf("%s failed to write stateless retry: %v", addr, err)
				} else {
					log.Printf("%s stateless retry: %x", addr, scid)
				}
				return
			}
			odcid = s.validateToken(addr, h.Token)
			if len(odcid) == 0 {
				log.Printf("%s invalid address validation token", addr)
				return
			}
			scid = h.DCID
		}
		c = serverConn{
			addr: addr,
			conn: quiche.Accept(scid, odcid, s.config),
		}
		s.conns[string(scid)] = c
		log.Printf("%s new connection: %x", addr, scid)
	}
	_, err = c.conn.Recv(buf)
	if err == quiche.ErrDone {
		return
	}
	if err != nil {
		log.Printf("%s failed to process packet: %v", addr, err)
		c.conn.Close(false, 0x1, []byte("fail"))
		return
	}
	if c.conn.IsEstablished() {
		s.recvStream(c.conn, buf[:cap(buf)])
	}
}

func (s *server) headerInfo(buf []byte, h *quiche.Header) error {
	h.SCID = h.SCID[:cap(h.SCID)]
	h.DCID = h.DCID[:cap(h.DCID)]
	h.Token = h.Token[:cap(h.Token)]
	return quiche.HeaderInfo(buf, quiche.MaxConnIDLen, h)
}

func (s *server) negotiate(addr net.Addr, h *quiche.Header, buf []byte) error {
	n, err := quiche.NegotiateVersion(h.SCID, h.DCID, buf)
	if err != nil {
		return err
	}
	_, err = s.socket.WriteTo(buf[:n], addr)
	return err
}

func (s *server) retry(addr net.Addr, h *quiche.Header, scid, buf []byte) error {
	token := s.mintToken(addr, h)
	n, err := quiche.Retry(h.SCID, h.DCID, scid, token, buf)
	if err != nil {
		return err
	}
	_, err = s.socket.WriteTo(buf[:n], addr)
	return err
}

// TODO: crypto
func (s *server) mintToken(addr net.Addr, h *quiche.Header) []byte {
	token := make([]byte, 0, maxTokenLen)
	token = append(token, []byte("quiche")...)
	token = append(token, []byte(addr.String())...)
	token = append(token, h.DCID...)
	return token
}

// TODO: crypto
func (s *server) validateToken(addr net.Addr, token []byte) []byte {
	if len(token) < 6 || string(token[:6]) != "quiche" {
		return nil
	}
	token = token[6:]
	addrStr := addr.String()
	if len(token) < len(addrStr) || string(token[:len(addrStr)]) != addrStr {
		return nil
	}
	return token[len(addrStr):]
}

func (s *server) recvStream(conn *quiche.Connection, buf []byte) {
	for {
		id, ok := conn.ReadableNext()
		if !ok {
			return
		}
		n, fin, err := conn.StreamRecv(id, buf)
		if err != nil {
			log.Printf("stream %d recv failed: %v", id, err)
			continue
		}
		log.Printf("stream %d has %d bytes (fin=%v)\n%s", id, n, fin, buf[:n])
		_, err = conn.StreamSend(id, []byte("Not Found"), true)
		if err != nil {
			log.Printf("stream send failed: %v", err)
			conn.Close(false, 0x1, []byte("fail"))
		}
	}
}

func (s *server) send(buf []byte) error {
	for _, c := range s.conns {
		n, err := c.conn.Send(buf)
		if err == quiche.ErrDone {
			log.Printf("%s done writing", c.addr)
			continue
		}
		if err != nil {
			log.Printf("%s send failed: %v", c.addr, err)
			c.conn.Close(false, 0x1, []byte("fail"))
			continue
		}
		n, err = s.socket.WriteTo(buf[:n], c.addr)
		if err != nil {
			return err
		}
		log.Printf("%s written %d bytes", c.addr, n)
	}
	return nil
}

func (s *server) close() {
	var stats quiche.Stats
	for k, c := range s.conns {
		if c.conn.IsClosed() {
			c.conn.Stats(&stats)
			log.Println("connection closed:", &stats)
			delete(s.conns, k)
			c.conn.Free()
		}
	}
}

func serverCommand(args []string) error {
	cmd := flag.NewFlagSet("server", flag.ExitOnError)
	verbose := cmd.Bool("v", false, "enable debug logging")
	listenAddr := cmd.String("listen", "127.0.0.1:4433", "listen on the given IP:port")
	certFile := cmd.String("cert", "cert.crt", "TLS certificate path")
	keyFile := cmd.String("key", "cert.key", "TLS certificate key path")
	rootPath := cmd.String("root", ".", "root directory")
	cmd.Parse(args)

	if *verbose {
		quiche.EnableDebugLogging()
	}
	config, err := newConfig(quiche.ProtocolVersion)
	if err != nil {
		return err
	}
	defer config.Free()
	if *certFile != "" {
		err = config.LoadCertChainFromPEMFile(*certFile)
		if err != nil {
			return err
		}
	}
	if *keyFile != "" {
		err = config.LoadPrivKeyFromPEMFile(*keyFile)
		if err != nil {
			return err
		}
	}
	return listen(config, *listenAddr, *rootPath)
}
