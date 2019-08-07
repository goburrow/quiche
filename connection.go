package quiche

/*
#include <stdlib.h>
#include <sys/types.h>
#include "quiche.h"
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"
)

// Connection is a QUIC connection.
type Connection C.quiche_conn

// Accept creates a new server-side connection.
func Accept(scid []byte, odcid []byte, config *Config) *Connection {
	scidp := cbytes(scid)
	// odcid is optional
	var odcidp *C.uint8_t
	if odcid != nil {
		odcidp = cbytes(odcid)
	}
	conn := C.quiche_accept(scidp, C.size_t(len(scid)),
		odcidp, C.size_t(len(odcid)),
		(*C.quiche_config)(config))

	return (*Connection)(conn)
}

// Connect creates a new server-side connection.
func Connect(serverName string, scid []byte, config *Config) *Connection {
	// serverName is optional
	var snp *C.char
	if serverName != "" {
		snp = C.CString(serverName)
	}
	scidp := cbytes(scid)

	conn := C.quiche_connect(snp,
		scidp, C.size_t(len(scid)),
		(*C.quiche_config)(config))
	if snp != nil {
		C.free(unsafe.Pointer(snp))
	}
	return (*Connection)(conn)
}

// Recv processes QUIC packets received from the peer.
func (c *Connection) Recv(b []byte) (int, error) {
	bp := cbytes(b)
	n := C.quiche_conn_recv((*C.quiche_conn)(c),
		bp, C.size_t(len(b)))
	if n < 0 {
		return 0, Error(n)
	}
	return int(n), nil
}

// Send writes a single QUIC packet to be sent to the peer.
func (c *Connection) Send(b []byte) (int, error) {
	bp := cbytes(b)
	n := C.quiche_conn_send((*C.quiche_conn)(c),
		bp, C.size_t(len(b)))
	if n < 0 {
		return 0, Error(n)
	}
	return int(n), nil
}

// StreamRecv reads contiguous data from a stream.
func (c *Connection) StreamRecv(streamID uint64, b []byte) (int, bool, error) {
	bp := cbytes(b)
	var fin C.bool
	n := C.quiche_conn_stream_recv((*C.quiche_conn)(c),
		C.uint64_t(streamID),
		bp, C.size_t(len(b)),
		&fin)
	if n < 0 {
		return 0, false, Error(n)
	}
	return int(n), bool(fin), nil
}

// StreamSend writes data to a stream.
func (c *Connection) StreamSend(streamID uint64, b []byte, fin bool) (int, error) {
	bp := cbytes(b)
	n := C.quiche_conn_stream_send((*C.quiche_conn)(c),
		C.uint64_t(streamID),
		bp, C.size_t(len(b)),
		C.bool(fin))
	if n < 0 {
		return 0, Error(n)
	}
	return int(n), nil
}

// Shutdown is the stream's side to shutdown.
type Shutdown int

const (
	ShutdownRead  = Shutdown(C.QUICHE_SHUTDOWN_READ)  // Stop receiving stream data.
	ShutdownWrite = Shutdown(C.QUICHE_SHUTDOWN_WRITE) // Stop sending stream data.
)

// StreamShutdown shuts down reading or writing from/to the specified stream.
func (c *Connection) StreamShutdown(streamID uint64, direction Shutdown, err uint64) error {
	n := C.quiche_conn_stream_shutdown((*C.quiche_conn)(c),
		C.uint64_t(streamID),
		C.enum_quiche_shutdown(direction),
		C.uint64_t(err))
	if n < 0 {
		return Error(n)
	}
	return nil
}

// StreamFinished returns true if all the data has been read from the specified stream.
func (c *Connection) StreamFinished(streamID uint64) bool {
	return bool(C.quiche_conn_stream_finished((*C.quiche_conn)(c), C.uint64_t(streamID)))
}

// ReadableNext fetches the next stream that has outstanding data to read. Returns false if
// there are no readable streams.
func (c *Connection) ReadableNext() (uint64, bool) {
	var streamID C.uint64_t
	next := C.quiche_readable_next((*C.quiche_conn)(c), &streamID)
	if next {
		return uint64(streamID), true
	}
	return 0, false
}

// Timeout returns the amount of time until the next timeout event, as nanoseconds.
func (c *Connection) Timeout() time.Duration {
	return time.Duration(C.quiche_conn_timeout_as_nanos((*C.quiche_conn)(c))) * time.Nanosecond
}

// OnTimeout processes a timeout event.
func (c *Connection) OnTimeout() {
	C.quiche_conn_on_timeout((*C.quiche_conn)(c))
}

// Close closes the connection with the given error and reason.
func (c *Connection) Close(app bool, errCode uint16, reason []byte) error {
	rp := cbytes(reason)
	n := C.quiche_conn_close((*C.quiche_conn)(c),
		C.bool(app),
		C.uint16_t(errCode),
		rp, C.size_t(len(reason)))
	if n < 0 {
		return Error(n)
	}
	return nil
}

// ApplicationProto returns the negotiated ALPN protocol.
func (c *Connection) ApplicationProto() []byte {
	var out *C.uint8_t
	var outLen C.size_t
	C.quiche_conn_application_proto((*C.quiche_conn)(c), &out, &outLen)
	if outLen <= 0 {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(out), C.int(outLen))
}

// IsEstablished returns true if the connection handshake is complete.
func (c *Connection) IsEstablished() bool {
	return bool(C.quiche_conn_is_established((*C.quiche_conn)(c)))
}

// IsClosed returns true if the connection is closed.
func (c *Connection) IsClosed() bool {
	return bool(C.quiche_conn_is_closed((*C.quiche_conn)(c)))
}

// Stats collects and returns statistics about the connection.
func (c *Connection) Stats() Stats {
	var s C.quiche_stats
	C.quiche_conn_stats((*C.quiche_conn)(c), &s)
	return Stats{
		Recv: uint64(s.recv),
		Sent: uint64(s.sent),
		Lost: uint64(s.lost),
		RTT:  time.Duration(s.rtt) * time.Nanosecond,
		CWnd: uint64(s.cwnd),
	}
}

// Free frees the connection object.
func (c *Connection) Free() {
	C.quiche_conn_free((*C.quiche_conn)(c))
}

// Stats is statistics about the connection.
type Stats struct {
	Recv uint64        // The number of QUIC packets received on this connection.
	Sent uint64        // The number of QUIC packets sent on this connection.
	Lost uint64        // The number of QUIC packets that were lost.
	RTT  time.Duration // The estimated round-trip time of the connection.
	CWnd uint64        // The size in bytes of the connection's congestion window.
}

func (s *Stats) String() string {
	return fmt.Sprintf("recv=%d sent=%d lost=%d rtt=%s",
		s.Recv, s.Sent, s.Lost, s.RTT)
}
