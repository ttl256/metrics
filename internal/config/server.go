package config

import (
	"flag"
	"fmt"
	"os"
)

type Server struct {
	Address string
}

func DefaultServer() *Server {
	return &Server{
		Address: "localhost:8080",
	}
}

func (a *Server) ApplyEnv() error {
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		a.Address = v
	}
	return nil
}

func (a *Server) ApplyFlags(fs *flag.FlagSet, args []string) error {
	addr := fs.String("a", a.Address, "server address to listen on")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing command line flags: %w", err)
	}
	a.Address = *addr
	return nil
}
