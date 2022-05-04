// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"fortio.org/fortio/periodic"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
	"io/ioutil"
	"log"
	"math"
	"os"
)

type fileList []string

func (i *fileList) String() string {
	return "my string representation"
}

func (i *fileList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var files fileList

type PXX struct {
	Value      float64
	Connection int
}

func main() {

	flag.Var(&files, "file", "file list")
	flag.Parse()

	p99 := make([]PXX, 0)
	p90 := make([]PXX, 0)

	for _, filePath := range files {
		item, err := readFile(filePath)
		if err != nil {
			log.Fatalln(err)
		}
		for _, percentile := range item.DurationHistogram.Percentiles {
			if percentile.Percentile == 99 {
				p99 = append(p99, PXX{
					Value:      percentile.Value,
					Connection: item.NumThreads,
				})
			}
			if percentile.Percentile == 90 {
				p90 = append(p90, PXX{
					Value:      percentile.Value,
					Connection: item.NumThreads,
				})
			}
		}
	}

	err := renderImage("p99", p99)
	if err != nil {
		log.Fatalln(err)
	}
	err = renderImage("p90", p90)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("============ TORNADO ============")
	fmt.Println("Successfully analyze test data.")
}

func readFile(path string) (*periodic.RunnerResults, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fortio file(%s) with error: %v", path, err)
	}

	res := new(periodic.RunnerResults)
	err = json.Unmarshal(raw, res)
	if err != nil {
		return nil, fmt.Errorf("unmarshal fortio result with error: %v", err)
	}
	return res, nil
}

func renderImage(title string, values []PXX) error {
	x := make([]float64, 0)
	y := make([]float64, 0)
	for i := 1; i <= len(values); i++ {
		x = append(x, float64(i))
	}

	xTicks := make([]chart.Tick, 0)
	yTicks := make([]chart.Tick, 0)

	for i, value := range values {
		xTicks = append(xTicks, chart.Tick{
			Value: float64(i + 1),
			Label: fmt.Sprintf("%v", value.Connection),
		})
		yTicks = append(yTicks, chart.Tick{
			Value: value.Value,
			Label: fmt.Sprintf("%0.2fms", value.Value*1000),
		})
		y = append(y, value.Value)
	}

	series := chart.ContinuousSeries{
		Style: chart.Style{
			StrokeColor: drawing.ColorFromHex("34A853"),
			StrokeWidth: 1,
			DotWidth:    4,
			DotColor:    drawing.ColorFromHex("34A853"),
		},
		YValues: y,
		XValues: x,
		XValueFormatter: func(v interface{}) string {
			typed := v.(float64)
			return fmt.Sprintf("%v", math.Floor(typed))
		},
	}
	graph := chart.Chart{
		Title: title,
		TitleStyle: chart.Style{
			TextHorizontalAlign: chart.TextHorizontalAlignLeft,
			FontSize:            16,
		},
		Background: chart.Style{
			Padding: chart.Box{
				Left: 50,
			},
		},
		Width:  500,
		Height: 400,
		XAxis: chart.XAxis{
			Name:  "Connections",
			Ticks: xTicks,
		},
		YAxis: chart.YAxis{
			Name:  "Latency,milliseconds",
			Ticks: yTicks,
		},
		Series: []chart.Series{
			series,
		},
	}
	f, err := os.Create(title + ".svg")
	if err != nil {
		return err
	}
	defer f.Close()
	err = graph.Render(chart.SVG, f)
	if err != nil {
		return err
	}
	return nil
}
