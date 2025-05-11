package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/atadzan/dist-arith-go/internal/worker"
	"github.com/atadzan/dist-arith-go/internal/worker/grpc/calc"
	"google.golang.org/grpc"
)

func main() {
	computingPower := 1
	if v := os.Getenv("COMPUTING_POWER"); v != "" {
		if cp, err := strconv.Atoi(v); err == nil {
			computingPower = cp
		}
	}

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := calc.NewCalcWorkerServiceClient(conn)
	for i := 0; i < computingPower; i++ {
		go worker.Worker(i, client)
	}

	fmt.Printf("Worker started with %d workers\n", computingPower)
	select {}
}
