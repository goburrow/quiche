package quiche

/*
#include <stdlib.h>
#include <sys/types.h>
#include "quiche.h"
*/
import "C"
import (
	"time"
	"unsafe"
)

// Config stores configuration shared between multiple connections.
type Config C.quiche_config

// NewConfig creates a config object with the given version.
func NewConfig(version uint32) *Config {
	c := C.quiche_config_new(C.uint32_t(version))
	if c == nil {
		panic("could not create config")
	}
	return (*Config)(c)
}

// LoadCertChainFromPEMFile configures the given certificate chain.
func (c *Config) LoadCertChainFromPEMFile(path string) error {
	cs := C.CString(path)
	err := C.quiche_config_load_cert_chain_from_pem_file((*C.quiche_config)(c), cs)
	C.free(unsafe.Pointer(cs))
	if err != 0 {
		return Error(err)
	}
	return nil
}

// LoadPrivKeyFromPEMFile configures the given private key.
func (c *Config) LoadPrivKeyFromPEMFile(path string) error {
	cp := C.CString(path)
	err := C.quiche_config_load_priv_key_from_pem_file((*C.quiche_config)(c), cp)
	C.free(unsafe.Pointer(cp))
	if err != 0 {
		return Error(err)
	}
	return nil
}

// VerifyPeer configures whether to verify the peer's certificate.
func (c *Config) VerifyPeer(v bool) {
	C.quiche_config_verify_peer((*C.quiche_config)(c), C.bool(v))
}

// Grease configures whether to send GREASE.
func (c *Config) Grease(v bool) {
	C.quiche_config_grease((*C.quiche_config)(c), C.bool(v))
}

// LogKeys enables logging of secrets.
func (c *Config) LogKeys() {
	C.quiche_config_log_keys((*C.quiche_config)(c))
}

// SetApplicationProtos configures the list of supported application protocols.
func (c *Config) SetApplicationProtos(protos []byte) error {
	cp := cbytes(protos)
	err := C.quiche_config_set_application_protos((*C.quiche_config)(c), cp, C.size_t(len(protos)))
	if err != 0 {
		return Error(err)
	}
	return nil
}

// SetIdleTimeout sets the `idle_timeout` transport parameter.
func (c *Config) SetIdleTimeout(v time.Duration) {
	C.quiche_config_set_idle_timeout((*C.quiche_config)(c), C.uint64_t(v/time.Millisecond))
}

// SetMaxPacketSize sets the `max_packet_size` transport parameter.
func (c *Config) SetMaxPacketSize(v uint64) {
	C.quiche_config_set_max_packet_size((*C.quiche_config)(c), C.uint64_t(v))
}

// SetInitialMaxData sets the `initial_max_data` transport parameter.
func (c *Config) SetInitialMaxData(v uint64) {
	C.quiche_config_set_initial_max_data((*C.quiche_config)(c), C.uint64_t(v))
}

// SetInitialMaxStreamDataBidiLocal sets the `initial_max_stream_data_bidi_local` transport parameter.
func (c *Config) SetInitialMaxStreamDataBidiLocal(v uint64) {
	C.quiche_config_set_initial_max_stream_data_bidi_local((*C.quiche_config)(c), C.uint64_t(v))
}

// SetInitialMaxStreamDataBidiRemote sets the `initial_max_stream_data_bidi_remote` transport parameter.
func (c *Config) SetInitialMaxStreamDataBidiRemote(v uint64) {
	C.quiche_config_set_initial_max_stream_data_bidi_remote((*C.quiche_config)(c), C.uint64_t(v))
}

// SetInitialMaxStreamDataUni sets the `initial_max_stream_data_uni` transport parameter.
func (c *Config) SetInitialMaxStreamDataUni(v uint64) {
	C.quiche_config_set_initial_max_stream_data_uni((*C.quiche_config)(c), C.uint64_t(v))
}

// SetInitialMaxStreamsBidi sets the `initial_max_streams_bidi` transport parameter.
func (c *Config) SetInitialMaxStreamsBidi(v uint64) {
	C.quiche_config_set_initial_max_streams_bidi((*C.quiche_config)(c), C.uint64_t(v))
}

// SetInitialMaxStreamsUni sets the `initial_max_streams_uni` transport parameter.
func (c *Config) SetInitialMaxStreamsUni(v uint64) {
	C.quiche_config_set_initial_max_streams_uni((*C.quiche_config)(c), C.uint64_t(v))
}

// SetAckDelayExponent sets the `ack_delay_exponent` transport parameter.
func (c *Config) SetAckDelayExponent(v uint64) {
	C.quiche_config_set_ack_delay_exponent((*C.quiche_config)(c), C.uint64_t(v))
}

// SetMaxAckDelay sets the `max_ack_delay` transport parameter.
func (c *Config) SetMaxAckDelay(v uint64) {
	C.quiche_config_set_max_ack_delay((*C.quiche_config)(c), C.uint64_t(v))
}

// DisableMigration sets the `disable_migration` transport parameter.
func (c *Config) DisableMigration(v bool) {
	C.quiche_config_set_disable_migration((*C.quiche_config)(c), C.bool(v))
}

// Free frees the config object.
func (c *Config) Free() {
	C.quiche_config_free((*C.quiche_config)(c))
}
