run-orchestrator:
	go run cmd/orchestrator.go --port=8090
run-worker:
	go run cmd/worker.go --orchestratorAddress=http://localhost:8090
build:
	GOOS=linux GOARCH=amd64 go build -o servly-api  ./cmd/main.go
