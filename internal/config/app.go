package config

import "flag"

type Application struct {
	Address string
}

func NewApplication() *Application {
	address := flag.String("a", "localhost:8080", "server address to listen on")
	flag.Parse()
	return &Application{Address: *address}
}
