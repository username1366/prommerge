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

func (pd *PromData) ParseMetricData(in string, extraLabels []string) []*PromMetric {
	var metrics []*PromMetric
	helpMap := make(map[string]string)
	typeMap := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(in))

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 6 && line[0:6] == "# HELP" {
			if pd.OmitMeta {
				continue
			}
			log.Debugf("Metadata help %v", line)
			matches := helpRe.FindStringSubmatch(line)
			if matches == nil || len(matches) < 2 {
				log.Warnf("No matches found for the help input %v", line)
			}
			helpMap[matches[1]] = matches[0]
			continue
		}
		if len(line) > 6 && line[0:6] == "# TYPE" {
			if pd.OmitMeta {
				continue
			}
			log.Debugf("Metadata type %v", line)
			matches := typeRe.FindStringSubmatch(line)
			if matches == nil || len(matches) < 2 {
				log.Warnf("No matches found for the type input %v", line)
			}
			typeMap[matches[1]] = matches[0]
			continue
		}

		p, err := pd.MetricParser(line, extraLabels)
		if err != nil {
			log.Errorf("%v", err)
		}
		p.Help = helpMap[p.Name]
		p.Type = typeMap[p.Name]
		metrics = append(metrics, p)
		log.Debugf("Metric: %+v", p)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading input:", err)
	}
	return metrics
}

func (pd *PromData) MetricParser(input string, extraLabels []string) (*PromMetric, error) {
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

	if pd.Sort {
		p.sort = fmt.Sprintf("%v%v", p.Name, p.LabelList)
	}
	return p, nil
}
