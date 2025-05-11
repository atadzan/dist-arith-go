run-orchestrator:
	JWT_SECRET=helloWorld go run cmd/orchestrator/main.go
run-worker:
	go run cmd/worker/main.go
generate-proto:
	protoc --go_out=. --go-grpc_out=. pkg/grpc/calc.proto