package prommerge

import (
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"regexp"
	"sync"
)

const (
	MetricReStr = `^([\w]+)(?:{(.+?)})? ([0-9.e+-]+)`
	LabelReStr  = `^([\w]+)="(.+)"`
	TypeReStr   = `^#\sTYPE\s(\w+)\s.+`
	HelpReStr   = `^#\sHELP\s(\w+)\s.+`
)

var (
	metricRe = regexp.MustCompile(MetricReStr)
	labelRe  = regexp.MustCompile(LabelReStr)
	typeRe   = regexp.MustCompile(TypeReStr)
	helpRe   = regexp.MustCompile(HelpReStr)
)

func NewPromData(promTargets []PromTarget) *PromData {
	pd := &PromData{
		PromTargets: promTargets,
	}
	return pd
}

type PromData struct {
	PromMetrics []*PromMetric
	PromTargets []PromTarget
}

type PromTarget struct {
	Url         string
	ExtraLabels []string
}

// CollectTargets fetches metrics from multiple URLs concurrently and combines them
func (pd *PromData) CollectTargets() error {
	var metrics []*PromMetric
	var wg sync.WaitGroup
	ch := make(chan *PromChanData, len(pd.PromTargets))

	for i, _ := range pd.PromTargets {
		wg.Add(1)
		go pd.PromTargets[i].FetchData(&wg, ch) // Start a goroutine for each URL
	}

	wg.Wait() // Wait for all fetch operations to complete
	close(ch) // Close the channel after all goroutines report they are done

	for result := range ch {
		if result.Err != nil {
			return result.Err
		}
		metrics = append(metrics, ParseMetricData(result.Data, result.ExtraLabels)...)
	}
	pd.PromMetrics = metrics
	pd.sortPromMetrics()

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
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
		//DisableCompression:  true,
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
	var output string
	var prevMetric string
	for _, m := range pd.PromMetrics {
		mStr := fmt.Sprintf("%v%v %v", m.Name, func() string {
			if len(m.LabelList) == 0 {
				return ""
			}
			var labelPairs string
			labelPairs = "{"
			for i := 0; i < len(m.LabelList); i += 2 {
				labelPairs = labelPairs + fmt.Sprintf("%v=\"%v\"", m.LabelList[i], m.LabelList[i+1])
				if i != len(m.LabelList)-2 {
					labelPairs = labelPairs + ","
				}
			}
			labelPairs = labelPairs + "}"
			return labelPairs
		}(), m.Value)
		if prevMetric != m.Name && (m.Help != "" || m.Type != "") {
			output = output + fmt.Sprintf("%v\n", m.Help)
			output = output + fmt.Sprintf("%v\n", m.Type)
		}
		output = output + fmt.Sprintf("%v\n", mStr)
		prevMetric = m.Name
	}
	return output
}
