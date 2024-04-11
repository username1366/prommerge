package prommerge

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"regexp"
	"sync"
)

const (
	MetricReStr = `^([\w]+)(?:{(.+?)})? ([0-9.e+-]+)`
	LabelReStr  = `^([\w]+)="(.+)"`
)

var (
	metricRe = regexp.MustCompile(MetricReStr)
	labelRe  = regexp.MustCompile(LabelReStr)
)

func NewPromData(targets []string) *PromData {
	pd := &PromData{
		Targets: targets,
	}
	return pd
}

type PromData struct {
	PromMetrics []*PromMetric
	Targets     []string
}

func (pd *PromData) CollectTargets() {
	pd.PromMetrics = combineMetrics(pd.Targets...)
	log.Printf("Merged %v metrics", len(pd.PromMetrics))
}

func (pd *PromData) ToString() string {
	var output string
	for _, m := range pd.PromMetrics {
		mStr := fmt.Sprintf("%v%v %v", m.Name, func() string {
			if len(m.Labels) == 0 {
				return ""
			}
			var labelPairs string
			labelPairs = "{"
			i := 0
			for k, v := range m.Labels {
				labelPairs = labelPairs + fmt.Sprintf("%v=\"%v\"", k, v)
				if i != len(m.Labels)-1 {
					labelPairs = labelPairs + ","
				}
				i++
			}
			labelPairs = labelPairs + "}"
			return labelPairs
		}(), m.Value)
		output = output + fmt.Sprintf("%v\n", mStr)
	}
	return output
}

// fetchData makes an HTTP GET request to the specified URL and sends the response body to a channel
func fetchData(url string, wg *sync.WaitGroup, ch chan<- *PromChanData) {
	defer wg.Done() // Signal that this goroutine is done after completing its task
	response, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error fetching data from %s: %v\n", url, err)
		ch <- nil // Send an empty string in case of error
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
		fmt.Printf("Error reading data from %s: %v\n", url, err)
		ch <- nil // Send an empty string in case of error
		return
	}
	ch <- &PromChanData{
		Data:   string(body),
		Source: url,
	}
}

type PromChanData struct {
	Data   string
	Source string
}

// combineMetrics fetches metrics from multiple URLs concurrently and combines them
func combineMetrics(urls ...string) []*PromMetric {
	var metrics []*PromMetric
	var wg sync.WaitGroup
	ch := make(chan *PromChanData, len(urls))

	for _, url := range urls {
		wg.Add(1)
		go fetchData(url, &wg, ch) // Start a goroutine for each URL
	}

	wg.Wait() // Wait for all fetch operations to complete
	close(ch) // Close the channel after all goroutines report they are done

	for result := range ch {
		metrics = append(metrics, Load(result.Data, result.Source)...)
	}
	return metrics
}
