package prommerge

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strings"
	"testing"
	"time"
)

func BenchmarkFunction(b *testing.B) {
	go http.ListenAndServe(":11112", promhttp.Handler())
	go http.ListenAndServe(":11113", promhttp.Handler())
	time.Sleep(time.Millisecond * 20)
	for i := 0; i < b.N; i++ {
		// call the function you want to test
		pd := NewPromData([]PromTarget{
			{
				Url: "http://127.1:11112/metrics",
				ExtraLabels: []string{
					`app="api"`,
					`source="internet"`,
				},
			},
			{
				Url: "http://127.1:11113/metrics",
				ExtraLabels: []string{
					`app="web"`,
				},
			},
		})
		pd.CollectTargets()
		_ = pd.ToString()
	}
}

// TestAdd tests the Add function.
func TestMerge(t *testing.T) {
	//log.SetLevel(log.DebugLevel)
	go http.ListenAndServe(":11112", promhttp.Handler())
	go http.ListenAndServe(":11113", promhttp.Handler())
	time.Sleep(time.Millisecond * 20)

	pd := NewPromData([]PromTarget{
		{
			Url: "http://127.1:11112/metrics",
			ExtraLabels: []string{
				`app="api"`,
				`source="internet"`,
			},
		},
		{
			Url: "http://127.1:11113/metrics",
			ExtraLabels: []string{
				`app="web"`,
			},
		},
	})
	pd.CollectTargets()
	result := pd.ToString()
	fmt.Printf("%v\n", result)

	expected := "promhttp_metric_handler_requests_total"
	if !strings.Contains(pd.ToString(), expected) {
		t.Errorf("Receive %v; want %v", result, expected)
	}
}
