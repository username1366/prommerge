package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/username1366/prommerge"
	"net/http"
	"time"
)

const (
	Socket = ":9393"
)

func main() {
	log.Printf("Listen server %v", Socket)
	http.HandleFunc("/prommerge", func(writer http.ResponseWriter, request *http.Request) {
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
		t := time.Now()
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
