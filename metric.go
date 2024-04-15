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
	Name      string
	LabelList []string
	Value     float64
	Help      string
	Type      string
	sort      string
}

func ParseMetricData(in string, extraLabels []string) []*PromMetric {
	var metrics []*PromMetric
	helpMap := make(map[string]string)
	typeMap := make(map[string]string)

	lines := ReadStringLineByLine(in)
	for i, _ := range lines {
		if len(lines[i]) > 6 && lines[i][0:6] == "# HELP" {
			log.Debugf("Metadata help %v", lines[i])
			matches := helpRe.FindStringSubmatch(lines[i])
			if matches == nil || len(matches) < 2 {
				log.Warnf("No matches found for the help input %v", lines[i])
			}
			helpMap[matches[1]] = matches[0]
			continue
		}
		if len(lines[i]) > 6 && lines[i][0:6] == "# TYPE" {
			log.Debugf("Metadata type %v", lines[i])
			matches := typeRe.FindStringSubmatch(lines[i])
			if matches == nil || len(matches) < 2 {
				log.Warnf("No matches found for the type input %v", lines[i])
			}
			typeMap[matches[1]] = matches[0]
			continue
		}

		p, err := MetricParser(lines[i], extraLabels)
		if err != nil {
			log.Errorf("%v", err)
		}
		p.Help = helpMap[p.Name]
		p.Type = typeMap[p.Name]
		metrics = append(metrics, p)
		log.Debugf("Metric: %+v", p)
	}
	return metrics
}

func MetricParser(input string, extraLabels []string) (*PromMetric, error) {
	p := new(PromMetric)
	matches := metricRe.FindStringSubmatch(input)
	if matches == nil {
		return nil, fmt.Errorf("no matches found for the input %v", input)
	}

	log.Debugf("Metric Name: %v", matches[1])
	p.Name = matches[1]
	log.Debugf("Labels: %v", matches[2])
	labelPairs := strings.Split(matches[2], ",")
	for _, labelPair := range extraLabels {
		kv := strings.Split(labelPair, "=")
		if len(kv) == 2 {
			p.LabelList = append(p.LabelList, kv[0], strings.ReplaceAll(kv[1], `"`, ""))
		}
	}

	// Parse labels
	for _, lPair := range labelPairs {
		m := labelRe.FindStringSubmatch(lPair)
		if m == nil {
			log.Debugf("Metric %v has no labels", p.Name)
			continue
		}
		if len(m) > 2 {
			p.LabelList = append(p.LabelList, m[1], m[2])
		}
	}
	value, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing float, %v", err)
	}
	log.Debugf("Value: %v", value)
	p.Value = value
	p.sort = fmt.Sprintf("%v%v", p.Name, p.LabelList)
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
