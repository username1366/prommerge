package prommerge

import (
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

func (pd *PromData) AsyncHTTP() error {
	t := time.Now()
	defer func() {
		pd.CollectTargetsDuration = time.Since(t)
		slog.Debug("Targets are collected", slog.String("duration", pd.CollectTargetsDuration.String()), slog.Int("len", len(pd.PromMetrics)))
	}()
	httpWg, parserWg, bodyData, workerPool :=
		new(sync.WaitGroup),
		new(sync.WaitGroup),
		make(chan *PromChanData),
		make(chan struct{}, pd.workerPoolSize)

	pd.PromMetrics = nil

	for i, _ := range pd.PromTargets {
		httpWg.Add(1)
		go pd.AHTTP(httpWg, bodyData, workerPool, pd.PromTargets[i])
	}

	go func() {
		slog.Debug("Waiting HTTP routines")
		httpWg.Wait()
		slog.Debug("Close bodyData channel")
		close(bodyData)
		slog.Debug("HTTP routines are done")
	}()

	defer func() {
		close(pd.PromMetricsStream)
		<-pd.MergeWorkerDoneHook
		slog.Debug("Release lock")
	}()

	go pd.MetricsMergeWorker()

	for {
		select {
		case promData, ok := <-bodyData:
			if !ok {
				slog.Debug("bodyData is closed", slog.Int("len(bodyData)", len(bodyData)), slog.Int("len(PromMetricsStream)", len(pd.PromMetricsStream)))
				tM := time.Now()
				parserWg.Wait()
				slog.Debug("Merge routine completed", slog.String("duration", time.Since(tM).String()))
				return nil
			}
			if promData == nil {
				if !pd.SupressErrors {
					slog.Warn("Empty promData")
				}
				continue
			}
			if promData.Err != nil && pd.EmptyOnFailure {
				slog.Debug("Return empty result")
				pd.PromMetrics = nil
				return fmt.Errorf("failed make async http request, %v", promData.Err)
			}
			if promData.Err != nil && !pd.EmptyOnFailure {
				if !pd.SupressErrors {
					slog.Warn("Skip promData error")
					slog.Error("Failed make async http request", slog.String("err", promData.Err.Error()))
				}
				continue
			}

			// Send metric to merge worker
			parserWg.Add(1)
			go pd.RouteMetric(parserWg, promData)
		}
	}
}

func (pd *PromData) RouteMetric(wg *sync.WaitGroup, promData *PromChanData) {
	defer func() {
		wg.Done()
	}()
	metrics := pd.ParseMetricData(promData.Data, promData.ExtraLabels)
	if metrics != nil {
		pd.PromMetricsStream <- metrics
	}
}

func (pd *PromData) MetricsMergeWorker() {
	for {
		select {
		case metrics, ok := <-pd.PromMetricsStream:
			if !ok {
				slog.Debug("Metrics stream is closed, all messages should be processed", slog.Int("len(PromMetricsStream)", len(pd.PromMetricsStream)))
				pd.MergeWorkerDoneHook <- struct{}{}
				return
			}
			pd.PromMetrics = append(pd.PromMetrics, metrics...)
		}
	}
}

func (pd *PromData) AHTTP(wg *sync.WaitGroup, bodyData chan *PromChanData, workerPool chan struct{}, target PromTarget) {
	defer wg.Done()
	defer func() {
		slog.Debug("Release worker")
		<-workerPool
	}()

	t := time.Now()
	slog.Debug("Acquire worker")
	workerPool <- struct{}{}

	slog.Debug("Get endpoint", slog.String("url", target.Url))
	response, err := pd.httpClient.Get(target.Url)
	if err != nil {
		bodyData <- &PromChanData{Err: fmt.Errorf("http get error for %s: %v", target.Url, err)}
		return
	}
	if response.StatusCode > 299 {
		bodyData <- &PromChanData{Err: fmt.Errorf("http get failed for %s, response code expected 200, actual %v", target.Url, response.StatusCode)}
		return
	}
	defer func() {
		err = response.Body.Close()
		if err != nil {
			slog.Error("Failed to close http request body", slog.String("err", err.Error()))
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		bodyData <- &PromChanData{Err: fmt.Errorf("error reading data from %s: %v", target.Url, err)}
		return
	}
	bodyData <- &PromChanData{
		Data:        string(body),
		ExtraLabels: target.ExtraLabels,
	}

	slog.Debug("Async http executed", slog.String("duration", time.Since(t).String()))
	return
}
