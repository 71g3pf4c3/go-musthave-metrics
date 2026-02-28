package main

import (
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/agent"
)

const (
	PollInterval   = 2
	ReportInterval = 10
)

func main() {
	a := agent.New("http://localhost:8080")

	go func() {
		for {
			a.Collect()
			time.Sleep(PollInterval * time.Second)
		}
	}()

	for {
		time.Sleep(ReportInterval * time.Second)
		a.Report()
	}
}
