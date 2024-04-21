package main

import (
	"fmt"
	"github.com/lmittmann/tint"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/username1366/prommerge"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const (
	BasePort       = 10000
	NumPromTargets = 3
	Socket         = ":9393"
)

func GetPromTargets() []prommerge.PromTarget {
	var targets []prommerge.PromTarget
	for i := 0; i < NumPromTargets; i++ {
		url := fmt.Sprintf("http://127.1:%v/metrics", BasePort+i)
		socket := fmt.Sprintf(":%v", BasePort+i)
		go func() {
			slog.Error("HTTP server error", http.ListenAndServe(socket, promhttp.Handler()).Error())
		}()
		targets = append(targets, prommerge.PromTarget{
			Url:         url,
			ExtraLabels: []string{fmt.Sprintf("app=%v", i)},
		})
	}
	return targets
}

func GetPromTargetsSingleServer() []prommerge.PromTarget {
	var targets []prommerge.PromTarget
	go func() {
		slog.Error("HTTP server error", http.ListenAndServe(fmt.Sprintf(":%v", BasePort), nil))
	}()
	for i := 0; i < NumPromTargets; i++ {
		url := fmt.Sprintf("http://127.1:%v/metrics%v", BasePort, i)
		endpoint := fmt.Sprintf("/metrics%v", i)
		http.Handle(endpoint, promhttp.Handler())

		targets = append(targets, prommerge.PromTarget{
			Url:         url,
			ExtraLabels: []string{fmt.Sprintf("app=%v", i)},
		})
	}
	return targets
}

func main() {
	w := os.Stderr
	// create a new logger
	logger := slog.New(tint.NewHandler(w, nil))

	// set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level: slog.LevelInfo,
			//TimeFormat: time.DateTime,
		}),
	))

	slog.Info("Listen server", slog.String("socket", Socket))
	getTargetsTime := time.Now()
	//targets := GetPromTargets()
	targets := GetPromTargetsSingleServer()
	slog.Info("Get targets generation is finished", slog.String("duration", time.Since(getTargetsTime).String()))
	http.HandleFunc("/prommerge", func(writer http.ResponseWriter, request *http.Request) {
		t := time.Now()
		pd := prommerge.NewPromData(targets, false, true, true, false)
		defer func() {
			logger.Info("Request finished", slog.String("duration", time.Since(t).String()))
		}()
		collectTargetsTime := time.Now()
		err := pd.CollectTargets()
		if err != nil {
			log.Errorf("%v", err)
		}
		logger.Info("Targets are collected", slog.String("duration", time.Since(collectTargetsTime).String()))
		outputTime := time.Now()
		output := pd.ToString()
		logger.Info("Output is generated", slog.String("duration", time.Since(outputTime).String()))
		writer.Write([]byte(output))
	})
	logger.Error("Listen error", http.ListenAndServe(Socket, nil))
}
