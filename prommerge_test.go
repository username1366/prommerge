package prommerge

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestAdd tests the Add function.
func TestMerge(t *testing.T) {

	go http.ListenAndServe(":11112", promhttp.Handler())
	go http.ListenAndServe(":11113", promhttp.Handler())
	time.Sleep(time.Millisecond * 40)

	pd := NewPromData([]string{
		"http://localhost:11112/metrics",
		"http://localhost:11113/metrics",
	})
	pd.CollectTargets()
	result := pd.ToString()
	fmt.Printf("%v\n", result)

	expected := "promhttp_metric_handler_requests_total"
	if !strings.Contains(pd.ToString(), expected) {
		t.Errorf("Receive %v; want %v", result, expected)
	}
}
