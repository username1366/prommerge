package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/username1366/prommerge"
	"log/slog"
	"net/http"
	"time"
)

const (
	BasePort       = 10000
	NumPromTargets = 200
	Socket         = ":9393"
)

func GetPromTargets() []prommerge.PromTarget {
	var targets []prommerge.PromTarget
	for i := 0; i < NumPromTargets; i++ {
		url := fmt.Sprintf("http://localhost:%v/metrics", BasePort+i)
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

func main() {
	log.Printf("Listen server %v", Socket)
	targets := GetPromTargets()
	http.HandleFunc("/prommerge", func(writer http.ResponseWriter, request *http.Request) {
		/*
			pd := prommerge.NewPromData([]prommerge.PromTarget{
				{
					Url: "http://127.1:8111/metrics",
					ExtraLabels: []string{
						`app="api"`,
						`source="internet"`,
					},
				},
				{
					Url: "http://127.1:8112/metrics",
					ExtraLabels: []string{
						`app="web"`,
					},
				},
			}, true, true, false, true)
		*/
		t := time.Now()
		pd := prommerge.NewPromData(targets, true, true, false, true)
		defer func() {
			log.Printf("Exec time %v", time.Since(t))
		}()
		err := pd.CollectTargets()
		if err != nil {
			log.Errorf("%v", err)
		}
		writer.Write([]byte(pd.ToString()))
	})
	http.ListenAndServe(Socket, nil)
}
