package main

import (
	"crypto/rand"
	"io"
	"log"
	"net"
	"time"

	"github.com/goburrow/quiche"
)

const maxDatagramSize = 1350
const httpRequestStreamID = 4

func createConfig() (*quiche.Config, error) {
	config := quiche.NewConfig(0xbabababa)
	err := config.SetApplicationProtos([]byte("\x05hq-20\x08http/0.9"))
	if err != nil {
		config.Free()
		return nil, err
	}
	config.SetIdleTimeout(5 * time.Second)
	config.SetMaxPacketSize(maxDatagramSize)
	config.SetInitialMaxData(10000000)
	config.SetInitialMaxStreamDataBidiLocal(1000000)
	config.SetInitialMaxStreamDataBidiRemote(1000000)
	config.SetInitialMaxStreamsBidi(100)
	config.SetInitialMaxStreamsUni(100)
	config.DisableMigration(true)
	config.VerifyPeer(false)
	return config, nil
}

func newConnID() []byte {
	b := make([]byte, quiche.MaxConnIDLen)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

func dial(addr string) (net.Conn, error) {
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

func recv(conn *quiche.Connection, socket io.Reader, buf []byte) error {
	for {
		n, err := socket.Read(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				log.Print("timed out")
				conn.OnTimeout()
				return nil
			}
			return err
		}
		log.Printf("got %d bytes", n)
		n, err = conn.Recv(buf[:n])
		if err == quiche.ErrDone {
			log.Print("done reading")
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func send(conn *quiche.Connection, socket io.Writer, buf []byte) error {
	for {
		n, err := conn.Send(buf)
		if err == quiche.ErrDone {
			log.Print("done writing")
			return nil
		}
		if err != nil {
			return err
		}
		n, err = socket.Write(buf[:n])
		if err != nil {
			return err
		}
		log.Printf("sent %d bytes", n)
	}
}

func recvStream(conn *quiche.Connection, buf []byte) {
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
		log.Printf("stream %d has %d bytes (fin? %v)", id, n, fin)
		if id == httpRequestStreamID {
			log.Printf("%s", buf[:n])
			log.Print("response received, closing...")
			conn.Close(true, 0x00, []byte("kthxbye"))
		}
	}
}

func connect() error {
	config, err := createConfig()
	if err != nil {
		return err
	}
	defer config.Free()
	socket, err := dial("127.0.0.1:4433")
	if err != nil {
		return err
	}
	defer socket.Close()

	buf := make([]byte, 65536)
	out := make([]byte, maxDatagramSize)

	scid := newConnID()
	conn := quiche.Connect("127.0.0.1", scid, config)
	defer conn.Free()
	n, err := conn.Send(out)
	if err != nil {
		return err
	}
	_, err = socket.Write(out[:n])
	if err != nil {
		return err
	}
	log.Printf("sent initial %d bytes", n)
	reqSent := false
	for {
		timeout := conn.Timeout()
		if timeout > 0 {
			err = socket.SetReadDeadline(time.Now().Add(timeout))
		} else {
			err = socket.SetReadDeadline(time.Time{})
		}
		if err != nil {
			return err
		}
		err = recv(conn, socket, buf)
		if err != nil {
			return err
		}
		if conn.IsClosed() {
			log.Print("connection closed")
			break
		}
		if conn.IsEstablished() && !reqSent {
			log.Print("sending HTTP request")
			_, err = conn.StreamSend(httpRequestStreamID, []byte("GET /\r\n"), true)
			if err != nil {
				return err
			}
			reqSent = true
		}
		recvStream(conn, buf)
		err := send(conn, socket, out)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	err := connect()
	if err != nil {
		log.Fatal(err)
	}
}
