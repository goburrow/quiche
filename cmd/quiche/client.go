package main

import (
	"flag"
	"log"
	"net"
	"strings"
	"time"

	"github.com/goburrow/quiche"
)

func hostname(addr string) string {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return addr
	}
	return addr[:idx]
}

func dialUDP(addr string) (net.Conn, error) {
	localAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		return nil, err
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", localAddr, remoteAddr)
}

func connect(config *quiche.Config, addr string) error {
	socket, err := dialUDP(addr)
	if err != nil {
		return err
	}
	defer socket.Close()

	scid := newConnID()
	conn := quiche.Connect(hostname(addr), scid, config)
	defer conn.Free()

	c := client{
		socket: socket,
		conn:   conn,
	}
	return c.connect()
}

type client struct {
	socket net.Conn
	conn   *quiche.Connection
}

func (c *client) connect() error {
	buf := make([]byte, bufferSize)
	err := c.send(buf)
	if err != nil {
		return err
	}
	log.Print("sent initial packet")
	reqSent := false
	for {
		err = c.recv(buf)
		if err != nil {
			return err
		}
		if c.conn.IsClosed() {
			log.Print("connection closed")
			return nil
		}
		if c.conn.IsEstablished() {
			if !reqSent {
				log.Print("sending HTTP request")
				_, err = c.conn.StreamSend(httpRequestStreamID, []byte("GET /\r\n"), true)
				if err != nil {
					return err
				}
				reqSent = true
			}
			c.recvStream(buf)
		}
		err = c.send(buf[:maxDatagramSize])
		if err != nil {
			return err
		}
	}
}

func (c *client) readDeadline() time.Time {
	timeout := c.conn.Timeout()
	if timeout > 0 {
		return time.Now().Add(timeout)
	}
	return time.Time{}
}

func (c *client) recv(buf []byte) error {
	deadline := c.readDeadline()
	err := c.socket.SetReadDeadline(deadline)
	if err != nil {
		return err
	}
	n, err := c.socket.Read(buf)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			log.Print("timed out")
			c.conn.OnTimeout()
			return nil
		}
		return err
	}
	_, err = c.conn.Recv(buf[:n])
	if err == quiche.ErrDone {
		return nil
	}
	return err
}

func (c *client) recvStream(buf []byte) {
	for {
		id, ok := c.conn.ReadableNext()
		if !ok {
			return
		}
		n, fin, err := c.conn.StreamRecv(id, buf)
		if err != nil {
			log.Printf("stream %d recv failed: %v", id, err)
			continue
		}
		log.Printf("stream %d has %d bytes (fin=%v)\n%s", id, n, fin, buf[:n])
		if id == httpRequestStreamID {
			log.Print("response received, closing...")
			c.conn.Close(true, 0x00, []byte("bye"))
		}
	}
}

func (c *client) send(buf []byte) error {
	for {
		n, err := c.conn.Send(buf)
		if err == quiche.ErrDone {
			return nil
		}
		if err != nil {
			return err
		}
		n, err = c.socket.Write(buf[:n])
		if err != nil {
			return err
		}
	}
}

func clientCommand(args []string) error {
	cmd := flag.NewFlagSet("client", flag.ExitOnError)
	verbose := cmd.Bool("v", false, "enable debug logging")
	wireVersion := cmd.Uint("wire-version", quiche.ProtocolVersion, "the version number to send to the server")
	noVerify := cmd.Bool("no-verify", false, "don't verify server's certificate")
	serverAddr := cmd.String("url", "127.0.0.1:4433", "QUIC server address")
	cmd.Parse(args)

	if *verbose {
		quiche.EnableDebugLogging()
	}
	config, err := newConfig(uint32(*wireVersion))
	if err != nil {
		return err
	}
	if *noVerify {
		config.VerifyPeer(false)
	}
	defer config.Free()

	return connect(config, *serverAddr)
}
