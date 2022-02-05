package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Arafatk/glot"
)

var (
	scaleForCells bool
)

type Measurement struct {
	Size int
	Time int
}

type StatResult struct {
	Command      string
	Measurements []Measurement `json:"Entries"`
}

type PlotEntry struct {
	filename   string
	legendName string
	*StatResult
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func parseArgsAndStatFiles() ([]PlotEntry, string) {
	outPath := flag.String("o", "plot.png", "path to outputted plot")
	flag.BoolVar(&scaleForCells, "xscale-cells", false, "scale by cells, not columns")
	flag.Parse()

	fmt.Println("outPath =", *outPath)

	entriesToPlot := make([]PlotEntry, 0, len(os.Args))
	for _, v := range flag.Args()[1:] {
		split := strings.Split(v, ",")
		if len(split) != 2 {
			fmt.Printf("number of comma-delimited values in '%s' is not equal to 2, skipping\n", v)
			continue
		}

		entry := PlotEntry{
			filename:   split[0],
			legendName: split[1],
		}
		if !fileExists(entry.filename) {
			fmt.Printf("file '%s' doesn't exist, skipping\n", entry.filename)
		}
		entriesToPlot = append(entriesToPlot, entry)
	}

	for i := 0; i < len(entriesToPlot); i++ {
		f, err := os.Open(entriesToPlot[i].filename)
		if err != nil {
			fmt.Println("error opening", entriesToPlot[i].filename)
			fmt.Println(err)
			os.Exit(1)
		}

		err = json.NewDecoder(f).Decode(&entriesToPlot[i].StatResult)
		if err != nil {
			fmt.Println("error decoding", entriesToPlot[i].filename)
			fmt.Println(err)
			os.Exit(1)
		}
	}

	return entriesToPlot, *outPath
}

func findMeasurementsMinOrMax(entries *[]Measurement, f func(old, cur Measurement) bool) Measurement {
	res := (*entries)[0]
	for _, v := range (*entries)[1:] {
		if f(v, res) {
			res = v
		}
	}
	return res
}

func findMeasurementRanges(m *[]Measurement) (int, int, int, int) {
	xMin := findMeasurementsMinOrMax(m,
		func(old, cur Measurement) bool { return old.Size < cur.Size }).Size
	xMax := findMeasurementsMinOrMax(m,
		func(old, cur Measurement) bool { return old.Size > cur.Size }).Size
	yMin := findMeasurementsMinOrMax(m,
		func(old, cur Measurement) bool { return old.Time < cur.Time }).Time
	yMax := findMeasurementsMinOrMax(m,
		func(old, cur Measurement) bool { return old.Time > cur.Time }).Time

	return xMin, xMax, yMin, yMax
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	entriesToPlot, outPath := parseArgsAndStatFiles()

	dimensions := 2
	persist := false
	debug := false
	plot, err := glot.NewPlot(dimensions, persist, debug)
	if err != nil {
		fmt.Println("error creating a plot")
		fmt.Println("error")
		os.Exit(1)
	}

	// Find and set graph ranges
	xMin, xMax, yMin, yMax := findMeasurementRanges(&entriesToPlot[0].Measurements)
	for _, plotEntry := range entriesToPlot[1:] {
		m := plotEntry.Measurements
		_xMin, _xMax, _yMin, _yMax := findMeasurementRanges(&m)
		xMin = min(xMin, _xMin)
		xMax = max(xMax, _xMax)
		yMin = min(yMin, _yMin)
		yMax = max(yMax, _yMax)
	}
	plot.SetXrange(xMin, xMax)
	plot.SetYrange(yMin, yMax)

	if scaleForCells {
		plot.SetXLabel("(Розмір матриці)^2")
	} else {
		plot.SetXLabel("Розмір матриці")
	}
	plot.SetYLabel("Час (наносекунди)")

	for _, plotEntry := range entriesToPlot {
		m := plotEntry.Measurements
		data := make([][]int, 2)
		data[0] = make([]int, len(m))
		data[1] = make([]int, len(m))
		for i := 0; i < len(m); i++ {
			data[0][i] = m[i].Size
			data[1][i] = m[i].Time
		}

		if scaleForCells {
			for i := 0; i < len(data); i++ {
				data[0][i] *= data[0][i]
			}
		}

		err = plot.AddPointGroup(
			plotEntry.legendName,
			"lines",
			data,
		)

		if err != nil {
			fmt.Println("error adding a point group to", plotEntry.legendName)
			fmt.Println(err)
			os.Exit(1)
		}
	}

	plot.SavePlot(outPath)
}
