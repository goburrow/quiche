package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/goburrow/quiche"
)

const maxDatagramSize = 1232
const bufferSize = 2048
const httpRequestStreamID = 4

func main() {
	flag.Usage = func() {
		output := flag.CommandLine.Output()
		fmt.Fprintln(output, "Usage: quiche (client|server) [options]")
		flag.PrintDefaults()
	}
	flag.Parse()
	cmd := flag.Arg(0)
	var err error
	switch cmd {
	case "client":
		err = clientCommand(flag.Args()[1:])
	case "server":
		err = serverCommand(flag.Args()[1:])
	default:
		flag.Usage()
		os.Exit(2)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func newConfig(version uint32) (*quiche.Config, error) {
	config := quiche.NewConfig(version)
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
