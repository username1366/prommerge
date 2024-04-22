package main

import (
	"fmt"
	"github.com/lmittmann/tint"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/username1366/prommerge"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

const (
	BasePort       = 10000
	NumPromTargets = 100
	Socket         = ":9393"
)

func GetPromTargets() []prommerge.PromTarget {
	var targets []prommerge.PromTarget
	for i := 0; i < NumPromTargets; i++ {
		url := fmt.Sprintf("http://127.1:%v/metrics", BasePort+i)
		socket := fmt.Sprintf(":%v", BasePort+i)
		go func() {
			slog.Error("HTTP server error", slog.String("err", http.ListenAndServe(socket, promhttp.Handler()).Error()))
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

	go http.ListenAndServe(":3333", promhttp.Handler())
	go http.ListenAndServe("localhost:6060", nil)

	for i := 0; i < NumPromTargets; i++ {
		url := fmt.Sprintf("http://127.1:%v/metrics%v", BasePort, i)
		targets = append(targets, prommerge.PromTarget{
			Url:         url,
			ExtraLabels: []string{fmt.Sprintf("app=%v", i)},
		})
	}
	return targets
}

func main() {
	//Pyroscope()
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
	httpClient := &http.Client{
		Timeout: time.Second * 30, // Set a total timeout for the request
		Transport: &http.Transport{
			MaxIdleConns:        500,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  true,
			MaxIdleConnsPerHost: 200,q
		},
	}
	slog.Info("Get targets generation is finished", slog.String("duration", time.Since(getTargetsTime).String()))
	http.HandleFunc("/prommerge", func(writer http.ResponseWriter, request *http.Request) {
		t := time.Now()
		pd := prommerge.NewPromData(targets, prommerge.PromDataOpts{
			EmptyOnFailure: false,
			Async:          true,
			Sort:           true,
			OmitMeta:       true,
			SupressErrors:  false,
			HTTPClient:     httpClient,
		})

		err := pd.CollectTargets()
		if err != nil {
			slog.Error("Failed to collect prometheus targets", slog.String("err", err.Error()))
		}
		output := pd.ToString()
		logger.Info("Request processed",
			slog.Duration("collect", pd.CollectTargetsDuration),
			slog.Duration("sort", pd.SortDuration),
			slog.Duration("out_prepare", pd.OutputPrepareDuration),
			slog.Duration("out_process", pd.OutputProcessDuration),
			slog.Duration("out_generate", pd.OutputGenerateDuration),
			slog.Duration("total_duration", time.Since(t)),
			slog.Int("total_metrics", len(pd.PromMetrics)),
		)
		writer.Write([]byte(output))
	})
	logger.Error("Listen error", slog.String("err", http.ListenAndServe(Socket, nil).Error()))
}
