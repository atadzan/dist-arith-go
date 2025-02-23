package handler

type Orchestrator interface{}

type orchestrator struct{}

func NewOrchestratorHandler() Orchestrator {
	return &orchestrator{}
}
