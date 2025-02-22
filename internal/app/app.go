package app

type App interface {
}

type app struct{}

func New() (App, error) {
	return nil, nil
}
