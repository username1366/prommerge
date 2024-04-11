package prommerge

import (
	"fmt"
	"sort"

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
	log.Debugf("Merged %v metrics", len(pd.PromMetrics))
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
	sortPromMetricsByName(metrics)
	return metrics
}

func sortPromMetricsByName(metrics []*PromMetric) {
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Name < metrics[j].Name
	})
}
