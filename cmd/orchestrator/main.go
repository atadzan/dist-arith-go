package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/atadzan/dist-arith-go/internal/repository"
	"github.com/atadzan/dist-arith-go/pkg/database"

	pb "github.com/atadzan/dist-arith-go/internal/worker/grpc/calc"

	"github.com/atadzan/dist-arith-go/internal/orchestrator"

	"google.golang.org/grpc"
)

const (
	httpPort     = ":8080"
	grpcPort     = ":50051"
	jwtSecretEnv = "JWT_SECRET"
)

func main() {
	fmt.Println("Orchestrator is running...")
	dbConn, err := database.NewDBConn("calc.db")
	if err != nil {
		log.Fatalf(err.Error())
	}
	repo, err := repository.New(dbConn)
	if err != nil {
		log.Fatalf("can't init db: %v", err)
	}
	defer dbConn.Close()

	if err = repo.CreateTables(); err != nil {
		log.Fatalf("migration err: %v", err)
	}

	jwtSecret := os.Getenv(jwtSecretEnv)
	if jwtSecret == "" {
		log.Fatalf("can't get %s from ENV", jwtSecretEnv)
	}
	authService := orchestrator.NewAuthService(repo, jwtSecret)
	schedulerService := orchestrator.NewScheduler(repo)
	grpcServerInstance := orchestrator.NewCalculatorGRPCServer(repo, schedulerService.GetOperationTimes(), schedulerService)

	httpHandlers := orchestrator.NewHTTPHandlers(authService, repo, schedulerService)

	go func() {
		lis, err := net.Listen("tcp", grpcPort)
		if err != nil {
			log.Fatalf("error while starting gRPC port %s: %v", grpcPort, err)
		}
		s := grpc.NewServer()
		pb.RegisterCalcWorkerServiceServer(s, grpcServerInstance)

		fmt.Printf("gRPC server listening %s\n", grpcPort)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	router := http.NewServeMux()

	router.HandleFunc("/api/v1/register", httpHandlers.RegisterHandler)
	router.HandleFunc("/api/v1/login", httpHandlers.LoginHandler)

	router.Handle("/api/v1/calculate", authService.JWTMiddleware(http.HandlerFunc(httpHandlers.CalculateHandler)))
	router.Handle("/api/v1/expressions", authService.JWTMiddleware(http.HandlerFunc(httpHandlers.ExpressionsHandler)))
	router.Handle("/api/v1/expressions/", authService.JWTMiddleware(http.HandlerFunc(httpHandlers.ExpressionsHandler)))

	fmt.Printf("HTTP is listening on port: %s\n", httpPort)
	corsRouter := orchestrator.EnableCORS(router)
	if err := http.ListenAndServe(httpPort, corsRouter); err != nil {
		log.Fatalf("Ошибка старта HTTP сервера: %v", err)
	}
	fmt.Println("Orchestrator stopped")
}
