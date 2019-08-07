package quiche

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

func randomCID() []byte {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

func defaultConfig() (*Config, error) {
	config := NewConfig(ProtocolVersion)
	err := config.LoadCertChainFromPEMFile("deps/quiche/examples/cert.crt")
	if err != nil {
		return nil, fmt.Errorf("load certificate: %v", err)
	}
	err = config.LoadPrivKeyFromPEMFile("deps/quiche/examples/cert.key")
	if err != nil {
		return nil, fmt.Errorf("load private key: %v", err)
	}
	err = config.SetApplicationProtos([]byte("\x06proto1\x06proto2"))
	if err != nil {
		return nil, fmt.Errorf("set application protocols: %v", err)
	}
	config.SetInitialMaxData(30)
	config.SetInitialMaxStreamDataBidiLocal(15)
	config.SetInitialMaxStreamDataBidiRemote(15)
	config.SetInitialMaxStreamDataUni(10)
	config.SetInitialMaxStreamsBidi(3)
	config.SetInitialMaxStreamsUni(3)
	config.VerifyPeer(false)
	return config, nil
}

func TestHandshake(t *testing.T) {
	// EnableDebugLogging()
	config, err := defaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	defer config.Free()
	client := Connect("", randomCID(), config)
	defer client.Free()
	server := Accept(randomCID(), nil, config)
	defer server.Free()

	buf := make([]byte, 65535)
	err = doHandshake(client, server, buf)
	if err != nil {
		t.Fatal(err)
	}
	if !client.IsEstablished() || !server.IsEstablished() {
		t.Fatalf("connection is not established: client=%v server=%v", client.IsEstablished(), server.IsEstablished())
	}
	clientProto := client.ApplicationProto()
	if string(clientProto) != "proto1" {
		t.Fatalf("unexpected protocol: client=%x", clientProto)
	}
	serverProto := server.ApplicationProto()
	if !bytes.Equal(clientProto, serverProto) {
		t.Fatalf("unmatched protocols: client=%x server=%x", clientProto, serverProto)
	}
	stats := client.Stats()
	t.Logf("client stats: %s", &stats)
	stats = server.Stats()
	t.Logf("server stats: %s", &stats)
}

func BenchmarkHandshake(b *testing.B) {
	config, err := defaultConfig()
	if err != nil {
		b.Fatal(err)
	}
	defer config.Free()
	clientCID := randomCID()
	serverCID := randomCID()
	buf := make([]byte, 65535)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := Connect("", clientCID, config)
		server := Accept(serverCID, nil, config)
		err := doHandshake(client, server, buf)
		if err != nil {
			b.Fatal(err)
		}
		server.Free()
		client.Free()
	}
}

func doHandshake(client, server *Connection, buf []byte) error {
	n, err := client.Send(buf)
	if err != nil {
		return err
	}
	for !client.IsEstablished() && !server.IsEstablished() {
		n, err = connRecvSend(server, buf, n)
		if err != nil {
			return err
		}
		n, err = connRecvSend(client, buf, n)
		if err != nil {
			return err
		}
	}
	n, err = connRecvSend(server, buf, n)
	if err != nil {
		return err
	}
	return nil
}

func connRecvSend(conn *Connection, b []byte, n int) (int, error) {
	left := n
	for left > 0 {
		i, err := conn.Recv(b[n-left : n])
		if err == ErrDone {
			break
		}
		if err != nil {
			return 0, err
		}
		left -= i
	}
	if left != 0 {
		return 0, fmt.Errorf("remaining should be 0: %d", left)
	}
	off := 0
	for off < len(b) {
		i, err := conn.Send(b[off:])
		if err == ErrDone {
			break
		}
		if err != nil {
			return 0, err
		}
		off += i
	}
	return off, nil
}
