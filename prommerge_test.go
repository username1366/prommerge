package prommerge

import (
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
	time.Sleep(time.Millisecond * 1)

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
	//fmt.Printf("%v\n", result)

	expectedList := []string{
		`go_threads{app="api",source="internet"}`,
		`go_threads{app="web"}`,
	}

	if !strings.Contains(pd.ToString(), expectedList[0]) {
		t.Errorf("Receive %v; want %v", result, expectedList[0])
	}
	if !strings.Contains(pd.ToString(), expectedList[1]) {
		t.Errorf("Receive %v; want %v", result, expectedList[1])
	}
}
