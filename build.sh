#!/bin/sh
set -eu

cmd() {
	cd cmd/quiche
	# Include libdl for static build
	#export CGO_LDFLAGS='-ldl'
	go build
	cd ../..
}

deps() {
	cd deps/quiche
	# Build only static library
	#sed -i 's/crate-type = ["lib", "staticlib", "cdylib"]/crate-type = ["lib", "staticlib"]/' Cargo.toml
	cargo build --release
	cd ../..
}

all() {
	deps
	cmd
}

if [ $# -lt 1 ]; then
	echo "$0" "(cmd|deps)"
	exit 2
fi

"$*"
