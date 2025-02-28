package main

import (
	"flag"
	"log"

	"github.com/atadzan/dist-arith-go/internal/server"
)

func main() {
	if err := server.Run(nil, getConfigPortsFromCli()); err != nil {
		log.Fatalln(err)
	}
}

func getConfigPortsFromCli() string {
	port := flag.String("port", "8080", "Default config")
	flag.Parse()
	return *port
}
