package prommerge

import (
	"fmt"
	"io"
	"log/slog"
	"sync"
)

const (
	basePort       = 14000
	workerPoolSize = 1
	numOfTargets   = 100
)

func (pd *PromData) AsyncHTTP() error {
	wg, bodyData, workerPool, done, errC :=
		new(sync.WaitGroup),
		make(chan *PromChanData),
		make(chan struct{}, workerPoolSize),
		make(chan struct{}),
		make(chan error)

	pd.PromMetrics = nil

	for _, target := range pd.PromTargets {
		wg.Add(1)
		go AHTTP(wg, target.Url, bodyData, target.ExtraLabels, workerPool, errC)
	}

	go func() {
		slog.Info("Waiting HTTP routines")
		wg.Wait()
		close(bodyData)
		close(errC)
		slog.Info("HTTP routines are done")
		done <- struct{}{}
	}()

	for {
		slog.Info("Inter")
		select {
		case promData := <-bodyData:
			if promData == nil {
				continue
			}
			if promData.Err != nil && pd.EmptyOnFailure {
				pd.PromMetrics = nil
				return fmt.Errorf("failed make async http request, %v", promData.Err)
			}
			if promData.Err != nil && !pd.EmptyOnFailure {
				slog.Error("Failed make async http request", slog.String("err", promData.Err.Error()))
				continue
			}
			pd.PromMetrics = append(pd.PromMetrics, pd.ParseMetricData(promData.Data, promData.ExtraLabels)...)
		case <-done:
			slog.Info("Request processed", slog.Int("len(bodyData)", len(bodyData)))
			return nil
		}
	}
}

func (pd *PromData) MetricsParserWorker(in string, labels []string) {
	pd.ParseMetricData(in, labels)
}

func (pd *PromData) MetricsMergeWorker(promMetrics chan []*PromMetric) {
	for {
		select {
		case metrics, ok := <-promMetrics:
			if !ok {
				return
			}
			pd.PromMetrics = append(pd.PromMetrics, metrics...)
		}
	}
}

func AHTTP(wg *sync.WaitGroup, url string, bodyData chan *PromChanData, extraLabels []string, workerPool chan struct{}, errC chan error) {
	defer wg.Done()
	defer func() {
		slog.Debug("Release worker")
		<-workerPool
	}()

	slog.Debug("Acquire worker")
	workerPool <- struct{}{}

	slog.Debug("Get endpoint", slog.String("url", url))
	response, err := httpClient.Get(url)
	if err != nil {
		//errC <- fmt.Errorf("http get error for %s: %v", url, err)
		bodyData <- &PromChanData{Err: fmt.Errorf("http get error for %s: %v", url, err)}
		return
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		//errC <- fmt.Errorf("error reading data from %s: %v", url, err)
		bodyData <- &PromChanData{Err: fmt.Errorf("error reading data from %s: %v", url, err)}
		return
	}
	bodyData <- &PromChanData{
		Data:        string(body),
		ExtraLabels: extraLabels,
	}
	return
}
