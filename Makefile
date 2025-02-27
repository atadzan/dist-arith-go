run:
	go run cmd/main.go --agent_port=8090 --orchestrator_port=8091
build:
	GOOS=linux GOARCH=amd64 go build -o servly-api  ./cmd/main.go