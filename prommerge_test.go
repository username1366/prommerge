package prommerge

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func BenchmarkSortEnabled(b *testing.B) {
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
		}, PromDataOpts{
			EmptyOnFailure: true,
			Async:          false,
			Sort:           true,
			OmitMeta:       true,
			SupressErrors:  false,
		})
		err := pd.CollectTargets()
		if err != nil {
			log.Fatal(err)
		}
		_ = pd.ToString()
	}
}

func BenchmarkSortDisabled(b *testing.B) {
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
		}, PromDataOpts{
			EmptyOnFailure: true,
			Async:          false,
			Sort:           false,
			OmitMeta:       true,
			SupressErrors:  false,
		})

		err := pd.CollectTargets()
		if err != nil {
			log.Fatal(err)
		}
		_ = pd.ToString()
	}
}

func BenchmarkMetaEnabled(b *testing.B) {
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
		}, PromDataOpts{
			EmptyOnFailure: true,
			Async:          false,
			Sort:           false,
			OmitMeta:       false,
			SupressErrors:  false,
		})

		err := pd.CollectTargets()
		if err != nil {
			log.Fatal(err)
		}
		_ = pd.ToString()
	}
}

func BenchmarkMetaDisabled(b *testing.B) {
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
		}, PromDataOpts{
			EmptyOnFailure: true,
			Async:          false,
			Sort:           false,
			OmitMeta:       true,
			SupressErrors:  false,
		})

		err := pd.CollectTargets()
		if err != nil {
			log.Fatal(err)
		}
		_ = pd.ToString()
	}
}
func Benchmark100Targets(b *testing.B) {
	basePort := 12000
	var targets []PromTarget
	for i := 0; i < 1000; i++ {
		go http.ListenAndServe(fmt.Sprintf(":%v", basePort+i), promhttp.Handler())
		targets = append(targets, PromTarget{
			Url: fmt.Sprintf("http://127.1:%v/metrics", basePort+i),
			ExtraLabels: []string{
				fmt.Sprintf(`app="api%v"`, i),
				`source="internet"`,
			},
		})
	}
	time.Sleep(time.Millisecond * 20)
	for i := 0; i < b.N; i++ {
		// call the function you want to test
		pd := NewPromData(targets, PromDataOpts{
			EmptyOnFailure: true,
			Async:          true,
			Sort:           false,
			OmitMeta:       true,
			SupressErrors:  false,
		})

		err := pd.CollectTargets()
		if err != nil {
			log.Fatal(err)
		}
		_ = pd.ToString()
	}
}

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
		}, PromDataOpts{
			EmptyOnFailure: true,
			Async:          false,
			Sort:           false,
			OmitMeta:       false,
			SupressErrors:  false,
		})

		err := pd.CollectTargets()
		if err != nil {
			log.Fatal(err)
		}
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
				`service="backend"`,
			},
		},
		{
			Url: "http://127.1:11113/metrics",
			ExtraLabels: []string{
				`app="web"`,
			},
		},
	}, PromDataOpts{
		EmptyOnFailure: false,
		Async:          false,
		Sort:           false,
		OmitMeta:       false,
		SupressErrors:  false,
	})

	err := pd.CollectTargets()
	if err != nil {
		log.Errorf("%v", err)
	}
	result := pd.ToString()
	if os.Getenv("DEBUG") != "" {
		fmt.Printf("%v\n", result)
	}

	expectedList := []string{
		`go_threads{app="api",source="internet",service="backend"}`,
		`go_threads{app="web"}`,
	}

	if !strings.Contains(pd.ToString(), expectedList[0]) {
		t.Errorf("Receive %v; want %v", result, expectedList[0])
	}
	if !strings.Contains(pd.ToString(), expectedList[1]) {
		t.Errorf("Receive %v; want %v", result, expectedList[1])
	}
}
