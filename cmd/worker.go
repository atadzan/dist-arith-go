package main

import (
	"log"

	"github.com/atadzan/dist-arith-go/internal/server"
)

func main() {
	if err := server.Run(nil, getConfigPortsFromCli()); err != nil {
		log.Fatalln(err)
	}
}
