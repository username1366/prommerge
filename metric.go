package prommerge

import (
	"bufio"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
)

type PromMetric struct {
	Name   string
	Labels map[string]string
	Value  float64
	Help   string
	Type   string
}

func Load(in string, source string) []*PromMetric {
	var metrics []*PromMetric
	lines := ReadStringLineByLine(in)
	for i, _ := range lines {
		if lines[i][0] == '#' {
			// Metadata
			log.Debugf("Metadata string %v", lines[i])
			continue
		}

		p, err := MetricParser(lines[i], source)
		if err != nil {
			log.Errorf("%v", err)
		}
		metrics = append(metrics, p)
		log.Debugf("Metric: %+v", p)
	}
	return metrics
}

func MetricParser(input, source string) (*PromMetric, error) {
	p := new(PromMetric)
	matches := metricRe.FindStringSubmatch(input)
	if matches == nil {
		return nil, fmt.Errorf("no matches found for the input %v", input)
	}

	log.Debugf("Metric Name: %v", matches[1])
	p.Name = matches[1]
	log.Debugf("Labels: %v", matches[2])
	labelPairs := strings.Split(matches[2], ",")
	p.Labels = make(map[string]string, len(labelPairs)+1)
	p.Labels["target_addr"] = source
	for _, lPair := range labelPairs {
		m := labelRe.FindStringSubmatch(lPair)
		if m == nil {
			log.Debugf("Metric %v has no labels", p.Name)
			continue
		}
		if len(m) > 2 {
			p.Labels[m[1]] = m[2]
		}
	}
	value, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing float, %v", err)
	}
	log.Debugf("Value: %v", value)
	p.Value = value
	return p, nil
}

// ReadStringLineByLine takes a multiline string and processes each line.
func ReadStringLineByLine(input string) []string {
	// Create a new scanner from the input string.
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(input))

	// Iterate over all lines in the input string.
	for scanner.Scan() {
		// Process the line, for example, by printing it.
		lines = append(lines, scanner.Text())

	}

	// Check for errors during Scan. End of file is expected and not reported by Scan as an error.
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading input:", err)
	}
	return lines
}
