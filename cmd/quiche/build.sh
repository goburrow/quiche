#/bin/sh
export CGO_LDFLAGS="${PWD}/../../deps/quiche/target/release/libquiche.a -ldl"
go build
