package handler

type Handler interface {
	Agent
	Orchestrator
}

type handler struct {
}

func New() Handler {
	return &handler{}
}
