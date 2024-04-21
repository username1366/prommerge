package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"net/http"
)

const (
	BasePort       = 10000
	NumPromTargets = 100
)

func main() {
	for i := 0; i < NumPromTargets; i++ {
		endpoint := fmt.Sprintf("/metrics%v", i)
		http.Handle(endpoint, promhttp.Handler())

	}
	socket := fmt.Sprintf(":%v", BasePort)
	slog.Info("Start http server", slog.String("socket", socket))
	slog.Error("HTTP server error", slog.String("err", http.ListenAndServe(socket, nil).Error()))
}
