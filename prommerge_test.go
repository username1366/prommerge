package prommerge

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"testing"
	"time"
)

// TestAdd tests the Add function.
func TestAdd(t *testing.T) {

	go http.ListenAndServe(":11112", promhttp.Handler())
	go http.ListenAndServe(":11113", promhttp.Handler())
	time.Sleep(time.Millisecond * 10)

	pd := NewPromData([]string{
		"http://localhost:11112/metrics",
		"http://localhost:11113/metrics",
	})
	pd.CollectTargets()
	//pd.ToString()
	fmt.Printf("%v\n", pd.ToString())

	result := 3
	expected := 3

	if result != expected {
		t.Errorf("Add(1, 2) = %d; want %d", result, expected)
	}
}
