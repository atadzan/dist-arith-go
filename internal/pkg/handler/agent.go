package handler

type Agent interface{}

type agent struct{}

func NewAgentHandler() Agent {
	return &agent{}
}
