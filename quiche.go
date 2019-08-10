// Package quiche provides Go bindings for Cloudflare Quiche.
package quiche

/*
#cgo CFLAGS: -Ideps/quiche/include
#cgo LDFLAGS: -Ldeps/quiche/target/release -lquiche

#include <stdio.h>
#include <sys/types.h>
#include "quiche.h"

static void debug_log(const char *line, void *argp) {
    fprintf(stderr, "%s\n", line);
}

static inline void log_to_stderr() {
	quiche_enable_debug_logging(debug_log, NULL);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// ProtocolVersion is the current QUIC wire version.
const ProtocolVersion = C.QUICHE_PROTOCOL_VERSION

// MaxConnIDLen is the maximum length of a connection ID.
const MaxConnIDLen = C.QUICHE_MAX_CONN_ID_LEN

// MinClientInitialLen is the minimum length of Initial packets sent by a client.
const MinClientInitialLen = C.QUICHE_MIN_CLIENT_INITIAL_LEN

// Error is a QUIC error.
type Error int

func (e Error) Error() string {
	if desc, ok := errorDescriptions[e]; ok {
		return desc
	}
	return fmt.Sprintf("unknown error (%d)", int(e))
}

const (
	ErrDone                  = Error(C.QUICHE_ERR_DONE)                    // There is no more work to do.
	ErrBufferTooShort        = Error(C.QUICHE_ERR_BUFFER_TOO_SHORT)        // The provided buffer is too short.
	ErrUnknownVersion        = Error(C.QUICHE_ERR_UNKNOWN_VERSION)         // The provided packet cannot be parsed because its version is unknown.
	ErrInvalidFrame          = Error(C.QUICHE_ERR_INVALID_FRAME)           // The provided packet cannot be parsed because it contains an invalid frame.
	ErrInvalidPacket         = Error(C.QUICHE_ERR_INVALID_PACKET)          // The provided packet cannot be parsed.
	ErrInvalidState          = Error(C.QUICHE_ERR_INVALID_STATE)           // The operation cannot be completed because the connection is in an invalid state.
	ErrInvalidStreamState    = Error(C.QUICHE_ERR_INVALID_STREAM_STATE)    // The operation cannot be completed because the stream is in an invalid state.
	ErrInvalidTransportParam = Error(C.QUICHE_ERR_INVALID_TRANSPORT_PARAM) // The peer's transport params cannot be parsed.
	ErrCryptoFail            = Error(C.QUICHE_ERR_CRYPTO_FAIL)             // A cryptographic operation failed.
	ErrTLSFail               = Error(C.QUICHE_ERR_TLS_FAIL)                // The TLS handshake failed.
	ErrFlowControl           = Error(C.QUICHE_ERR_FLOW_CONTROL)            // The peer violated the local flow control limits.
	ErrStreamLimit           = Error(C.QUICHE_ERR_STREAM_LIMIT)            // The peer violated the local stream limits.
	ErrFinalSize             = Error(C.QUICHE_ERR_FINAL_SIZE)              // The received data exceeds the stream's final size.
)

var errorDescriptions = map[Error]string{
	ErrDone:                  "nothing else to do",
	ErrBufferTooShort:        "buffer is too short",
	ErrUnknownVersion:        "version is unknown",
	ErrInvalidFrame:          "frame is invalid",
	ErrInvalidPacket:         "packet is invalid",
	ErrInvalidState:          "connection state is invalid",
	ErrInvalidStreamState:    "stream state is invalid",
	ErrInvalidTransportParam: "transport parameter is invalid",
	ErrCryptoFail:            "crypto operation failed",
	ErrTLSFail:               "TLS failed",
	ErrFlowControl:           "flow control limit was violated",
	ErrStreamLimit:           "stream limit was violated",
	ErrFinalSize:             "data exceeded stream's final size",
}

// Header is a QUIC packet's header.
type Header struct {
	Type    uint8
	Version uint32
	SCID    []byte
	DCID    []byte
	Token   []byte
}

// HeaderInfo extracts version, type, source / destination connection ID and address
// verification token from the packet in b.
func HeaderInfo(b []byte, dcidLength int, header *Header) error {
	var ty C.uint8_t
	var version C.uint32_t
	scidLen := clen(header.SCID)
	dcidLen := clen(header.DCID)
	tokenLen := clen(header.Token)
	n := C.quiche_header_info(cbytes(b), clen(b),
		C.size_t(dcidLength),
		&version, &ty,
		cbytes(header.SCID), &scidLen,
		cbytes(header.DCID), &dcidLen,
		cbytes(header.Token), &tokenLen)
	if n < 0 {
		if n == -1 {
			return ErrBufferTooShort
		}
		return Error(n)
	}
	header.Type = uint8(ty)
	header.Version = uint32(version)
	header.SCID = header.SCID[:scidLen]
	header.DCID = header.DCID[:dcidLen]
	header.Token = header.Token[:tokenLen]
	return nil
}

// NegotiateVersion writes a version negotiation packet.
func NegotiateVersion(scid, dcid, b []byte) (int, error) {
	n := C.quiche_negotiate_version(cbytes(scid), clen(scid),
		cbytes(dcid), clen(dcid),
		cbytes(b), clen(b))
	if n < 0 {
		return 0, Error(n)
	}
	return int(n), nil
}

// Retry writes a retry packet.
func Retry(scid, dcid, newSCID, token, b []byte) (int, error) {
	n := C.quiche_retry(cbytes(scid), clen(scid),
		cbytes(dcid), clen(dcid),
		cbytes(newSCID), clen(newSCID),
		cbytes(token), clen(token),
		cbytes(b), clen(b))
	if n < 0 {
		return 0, Error(n)
	}
	return int(n), nil
}

// EnableDebugLogging enables logging.
func EnableDebugLogging() {
	C.log_to_stderr()
}

var emptySlice = []byte{0}

func cbytes(s []byte) *C.uint8_t {
	if len(s) == 0 {
		s = emptySlice
	}
	return (*C.uint8_t)(unsafe.Pointer(&s[0]))
}

func clen(s []byte) C.size_t {
	return C.size_t(len(s))
}
