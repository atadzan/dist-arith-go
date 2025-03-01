package main

import (
	"flag"
	"log"

	"github.com/atadzan/dist-arith-go/internal/delivery"
	"github.com/atadzan/dist-arith-go/internal/server"
)

func main() {
	orchestratorHandler := delivery.NewOrchestratorHandler()
	orchestratorHandler.InitRoutes()
	if err := server.Run(orchestratorHandler.Handler, getOrchestratorConfigPortsFromCli()); err != nil {
		log.Fatalln(err)
	}
}

func getOrchestratorConfigPortsFromCli() string {
	port := flag.String("port", "8080", "Default config")
	flag.Parse()
	return *port
}
