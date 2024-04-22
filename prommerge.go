package prommerge

import (
	"bytes"
	"fmt"
	"log/slog"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"regexp"
	"sync"
)

const (
	MetricReStr           = `^([\w]+)(?:{(.+?)})? ([0-9.e+-]+)`
	LabelReStr            = `^([\w]+)="(.+)"`
	TypeReStr             = `^#\sTYPE\s(\w+)\s.+`
	HelpReStr             = `^#\sHELP\s(\w+)\s.+`
	DefaultWorkerPoolSize = 100
)

var (
	metricRe = regexp.MustCompile(MetricReStr)
	labelRe  = regexp.MustCompile(LabelReStr)
	typeRe   = regexp.MustCompile(TypeReStr)
	helpRe   = regexp.MustCompile(HelpReStr)
)

type PromDataOpts struct {
	EmptyOnFailure bool
	Async          bool
	Sort           bool
	OmitMeta       bool
	SupressErrors  bool
}

func NewPromData(promTargets []PromTarget, opts PromDataOpts) *PromData {
	pd := &PromData{
		PromTargets:         promTargets,
		PromMetricsStream:   make(chan []*PromMetric, 20),
		MergeWorkerDoneHook: make(chan struct{}),
		EmptyOnFailure:      opts.EmptyOnFailure,
		Async:               opts.Async,
		Sort:                opts.Sort,
		OmitMeta:            opts.OmitMeta,
		SupressErrors:       opts.SupressErrors,
		workerPoolSize: func() int {
			if opts.Async {
				return DefaultWorkerPoolSize
			}
			return 1
		}(),
	}
	return pd
}

type PromData struct {
	PromMetrics            []*PromMetric
	PromTargets            []PromTarget
	PromMetricsStream      chan []*PromMetric
	PromMetricsOutStream   chan string
	MergeWorkerDoneHook    chan struct{}
	CollectTargetsDuration time.Duration
	SortDuration           time.Duration
	OutputPrepareDuration  time.Duration
	OutputProcessDuration  time.Duration
	OutputGenerateDuration time.Duration
	workerPoolSize         int
	EmptyOnFailure         bool
	Async                  bool
	Sort                   bool
	OmitMeta               bool
	SupressErrors          bool
}

type PromTarget struct {
	Name        string
	Url         string
	ExtraLabels []string
}

// CollectTargets fetches metrics from multiple URLs concurrently and combines them
func (pd *PromData) CollectTargets() error {
	err := pd.AsyncHTTP()
	if err != nil {
		return err
	}
	if pd.Sort {
		if pd.OmitMeta {
			slog.Debug("Meta collecting is disabled, sort may not work")
		}
		t := time.Now()
		pd.sortPromMetrics()
		pd.SortDuration = time.Since(t)
		slog.Debug("Metrics sorted", slog.String("duration", pd.SortDuration.String()))
	}
	return nil
}

func (pd *PromData) sortPromMetrics() {
	sort.Slice(pd.PromMetrics, func(i, j int) bool {
		return pd.PromMetrics[i].sort < pd.PromMetrics[j].sort
		//return pd.PromMetrics[i].Name < pd.PromMetrics[j].Name
	})
}

// httpClient is a shared http.Client with a custom Transport
var httpClient = &http.Client{
	Timeout: time.Second * 30, // Set a total timeout for the request
	Transport: &http.Transport{
		MaxIdleConns:       100,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	},
}

// FetchData makes an HTTP GET request to the specified URL and sends the response body to a channel
func (pt *PromTarget) FetchData(wg *sync.WaitGroup, ch chan<- *PromChanData) {
	defer wg.Done() // Signal that this goroutine is done after completing its task
	response, err := httpClient.Get(pt.Url)
	if err != nil {
		ch <- &PromChanData{Err: fmt.Errorf("error fetching data from %s: %v", pt.Url, err)}
		return
	}
	defer func() {
		err = response.Body.Close()
		if err != nil {
			log.Errorf("Failed to close response body %v", err)
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		ch <- &PromChanData{Err: fmt.Errorf("error reading data from %s: %v", pt.Url, err)}
		return
	}
	ch <- &PromChanData{
		Data:        string(body),
		Source:      pt.Url,
		ExtraLabels: pt.ExtraLabels,
	}
}

type PromChanData struct {
	Data        string
	Source      string
	ExtraLabels []string
	Err         error
}

func (pd *PromData) ToString() string {
	var prevMetric string
	var buffer bytes.Buffer

	tP := time.Now()
	wg := &sync.WaitGroup{}
	workerPool := make(chan struct{}, 900)
	for n, _ := range pd.PromMetrics {
		wg.Add(1)
		workerPool <- struct{}{}
		go func(wg *sync.WaitGroup) {
			defer func() {
				<-workerPool
				wg.Done()
			}()
			pd.PromMetrics[n].Output = pd.BuildMetricString(n)
		}(wg)
	}
	wg.Wait()
	pd.OutputPrepareDuration = time.Since(tP)
	slog.Debug("Output is prepared", slog.String("duration", pd.OutputPrepareDuration.String()))

	t := time.Now()
	for n, _ := range pd.PromMetrics {
		// Process metadata
		if prevMetric != pd.PromMetrics[n].Name && (pd.PromMetrics[n].Help != "" || pd.PromMetrics[n].Type != "") {
			buffer.WriteString(pd.PromMetrics[n].Help)
			buffer.WriteString("\n")
			buffer.WriteString(pd.PromMetrics[n].Type)
			buffer.WriteString("\n")
		}
		tB := time.Now()
		buffer.WriteString(pd.PromMetrics[n].Output)
		slog.Debug("Processed output string", slog.String("duration", time.Since(tB).String()))
		prevMetric = pd.PromMetrics[n].Name
	}
	pd.OutputProcessDuration = time.Since(t)
	slog.Debug("Output processed", slog.Int("lines", len(pd.PromMetrics)), slog.String("duration", pd.OutputProcessDuration.String()))

	tB := time.Now()
	defer func() {
		pd.OutputGenerateDuration = time.Since(tB)
	}()
	defer func() {
		buffer.Reset()
	}()
	return buffer.String()
}

func (pd *PromData) BuildMetricString(n int) string {
	mStr := fmt.Sprintf("%v%v %v\n", pd.PromMetrics[n].Name, func() string {
		if len(pd.PromMetrics[n].LabelList) == 0 {
			return ""
		}
		var labelPairs string
		labelPairs = "{"
		for i := 0; i < len(pd.PromMetrics[n].LabelList); i += 2 {
			labelPairs = labelPairs + pd.PromMetrics[n].LabelList[i] + `="` + pd.PromMetrics[n].LabelList[i+1] + `"`
			if i != len(pd.PromMetrics[n].LabelList)-2 {
				labelPairs = labelPairs + ","
			}
		}
		labelPairs = labelPairs + "}"
		return labelPairs
	}(), pd.PromMetrics[n].Value)

	return mStr
}
