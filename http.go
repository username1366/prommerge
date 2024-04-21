package prommerge

import (
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

const (
	workerPoolSize = 100
)

func (pd *PromData) AsyncHTTP() error {
	httpWg, parserWg, bodyData, workerPool, done :=
		new(sync.WaitGroup),
		new(sync.WaitGroup),
		make(chan *PromChanData),
		make(chan struct{}, workerPoolSize),
		make(chan struct{})

	pd.PromMetrics = nil

	for _, target := range pd.PromTargets {
		httpWg.Add(1)
		go AHTTP(httpWg, target.Url, bodyData, target.ExtraLabels, workerPool)
	}

	go func() {
		slog.Debug("Waiting HTTP routines")
		httpWg.Wait()
		slog.Warn("Close bodyData channel")
		close(bodyData)
		slog.Debug("HTTP routines are done")
		done <- struct{}{}
	}()

	//time.Sleep(time.Second * 1)
	for {
		select {
		case promData, ok := <-bodyData:
			if !ok {
				slog.Warn("bodyData is closed", slog.Int("len(bodyData)", len(bodyData)), slog.Int("len(PromMetricsStream)", len(pd.PromMetricsStream)))
				t := time.Now()
				parserWg.Wait()
				slog.Info("Merge routine completed", slog.String("duration", time.Since(t).String()))
				close(pd.PromMetricsStream)
				<-pd.MergeWorkerDoneHook
				slog.Info("Release lock")
				return nil
			}
			if promData == nil {
				slog.Info("Empty promData")
				continue
			}
			if promData.Err != nil && pd.EmptyOnFailure {
				slog.Info("Return empty result")
				pd.PromMetrics = nil
				return fmt.Errorf("failed make async http request, %v", promData.Err)
			}
			if promData.Err != nil && !pd.EmptyOnFailure {
				slog.Info("Skip promData error")
				slog.Error("Failed make async http request", slog.String("err", promData.Err.Error()))
				continue
			}

			// Send metric to merge worker
			parserWg.Add(1)
			go func(wg *sync.WaitGroup) {
				defer func() {
					wg.Done()
				}()
				pd.PromMetricsStream <- pd.ParseMetricData(promData.Data, promData.ExtraLabels)
			}(parserWg)

			/*
				case <-done:
					slog.Info("Request processed", slog.Int("len(bodyData)", len(bodyData)), slog.Int("len(PromMetricsStream)", len(pd.PromMetricsStream)))
					t := time.Now()
					parserWg.Wait()
					slog.Info("All parsers executed", slog.String("duration", time.Since(t).String()))
					close(pd.PromMetricsStream)

					return nil

			*/
		}
	}
}

func (pd *PromData) MetricsMergeWorker() {
	for {
		select {
		case metrics, ok := <-pd.PromMetricsStream:
			if !ok {
				slog.Warn("Metrics stream is closed, all messages should be processed", slog.Int("len(PromMetricsStream)", len(pd.PromMetricsStream)))
				pd.MergeWorkerDoneHook <- struct{}{}
				slog.Warn("Hook!")
				return
			}
			pd.PromMetrics = append(pd.PromMetrics, metrics...)
		}
	}
}

func AHTTP(wg *sync.WaitGroup, url string, bodyData chan *PromChanData, extraLabels []string, workerPool chan struct{}) {
	defer wg.Done()
	defer func() {
		slog.Debug("Release worker")
		<-workerPool
	}()

	t := time.Now()

	slog.Debug("Acquire worker")
	workerPool <- struct{}{}

	slog.Debug("Get endpoint", slog.String("url", url))
	response, err := httpClient.Get(url)
	if err != nil {
		bodyData <- &PromChanData{Err: fmt.Errorf("http get error for %s: %v", url, err)}
		return
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		bodyData <- &PromChanData{Err: fmt.Errorf("error reading data from %s: %v", url, err)}
		return
	}
	bodyData <- &PromChanData{
		Data:        string(body),
		ExtraLabels: extraLabels,
	}

	slog.Debug("Async http executed", slog.String("duration", time.Since(t).String()))
	return
}
