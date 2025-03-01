package main

import (
	"flag"
	"log"

	"github.com/atadzan/dist-arith-go/internal/delivery"
)

func main() {
	workerHandler := delivery.NewWorkerHandler(getWorkerConfigFromCli())
	log.Println("Running worker...")
	workerHandler.Run()
}

func getWorkerConfigFromCli() string {
	port := flag.String("orchestratorAddress", "http://localhost:8080", "Default config")
	flag.Parse()
	return *port
}
