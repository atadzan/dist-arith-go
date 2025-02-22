package main

import "flag"

func main() {

}

func getConfigPortsFromCli() (string, string) {
	agentPort := flag.String("agent_port", "8080", "Default config")
	orchestratorPort := flag.String("orchestrator_port", "8081", "Default config")
	flag.Parse()
	return *agentPort, *orchestratorPort
}
