package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	matrixSize    int
	printMatrices bool
	concurrent    bool
	threadCount   uint

	statConfiguration     bool
	statStart             uint
	statEnd               uint
	statStep              uint
	statIterationsPerStep uint
	outputPath            string

	timePrecision func(time.Duration) int64
	timeSuffix    string

	forEachMatrixElementIndex func(func(int, int))

	_rand rand.Rand
)

func init() {
	// Parsing configuration
	flag.BoolVar(&printMatrices, "p", false, "print input matrices (works only in runOnce mode)")
	uintMatrixSize := flag.Uint("size", 3, "size of the input matrices (works only in runOnce mode)")
	flag.BoolVar(&concurrent, "concurrent", false, "run operations concurrently")
	flag.UintVar(&threadCount, "threads", 0, "number of threads excluding a producer thread")

	flag.BoolVar(&statConfiguration, "stat", false, "run in stat mode and output statistics as json")
	flag.UintVar(&statStart, "stat-start", 100, "where to start statting")
	flag.UintVar(&statEnd, "stat-end", 1000, "where to stop statting")
	flag.UintVar(&statStep, "stat-step", 100, "statting step")
	flag.UintVar(&statIterationsPerStep, "iterations", 5, "how many iterations to do each step")
	flag.StringVar(&outputPath, "o", "", "stat output file location (is constructed from parameters if not specified)")

	outputNanoseconds := flag.Bool("nano", false, "output time in nanoseconds (default is milliseconds)")

	printHelp := flag.Bool("help", false, "print help")

	flag.Parse()

	// Configuring based on parsed values
	if *printHelp {
		flag.Usage()
		os.Exit(0)
	}

	// General configurations
	matrixSize = int(*uintMatrixSize)

	if *outputNanoseconds {
		timePrecision = func(d time.Duration) int64 { return d.Nanoseconds() }
		timeSuffix = "ns"
	} else {
		timePrecision = func(d time.Duration) int64 { return d.Milliseconds() }
		timeSuffix = "ms"
	}

	// Concurrency settings
	if concurrent && threadCount == 0 {
		fmt.Println("Can't be concurrent with a thread count of 0, setting thread count to 3")
		threadCount = 3
	}

	if threadCount > 0 && !concurrent {
		concurrent = true
	}

	if concurrent {
		forEachMatrixElementIndex = func(f func(int, int)) {
			forEachEmbeddedIterationConcurrent(matrixSize, matrixSize, f)
		}
	} else {
		forEachMatrixElementIndex = func(f func(int, int)) {
			forEachEmbeddedIteration(matrixSize, matrixSize, f)
		}
	}

	// setting rand seed
	randSource := rand.NewSource(time.Now().UnixNano())
	_rand = *rand.New(randSource)

	// Summary
	fmt.Printf("matrix size: %dx%d\n", matrixSize, matrixSize)
	fmt.Println("time is in", timeSuffix)

	if concurrent {
		fmt.Println("thread count:", threadCount)
	} else {
		fmt.Println("operations will be run sequentially")
	}

	fmt.Println()
}

func makeSquareMatrix(size int) *[][]int {
	mat := make([][]int, size)
	for i := 0; i < size; i++ {
		mat[i] = make([]int, size)
	}

	return &mat
}

func forEachEmbeddedIteration(iMax, jMax int, f func(int, int)) {
	for i := 0; i < iMax; i++ {
		for j := 0; j < jMax; j++ {
			f(i, j)
		}
	}
}

func forEachEmbeddedIterationConcurrent(iMax, jMax int, f func(int, int)) {
	channel := make(chan int)

	var wg sync.WaitGroup
	wg.Add(int(threadCount))

	worker := func() {
		defer wg.Done()
		for {
			i, moreWork := <-channel
			if !moreWork {
				return
			}
			for j := 0; j < jMax; j++ {
				f(i, j)
			}
		}
	}

	for i := 0; i < int(threadCount); i++ {
		go worker()
	}

	for i := 0; i < iMax; i++ {
		channel <- i
	}
	close(channel)

	wg.Wait()
}

func fillMatrix(m *[][]int) {
	kindaSeed := _rand.Intn(100)
	forEachMatrixElementIndex(
		func(i, j int) {
			(*m)[i][j] = kindaSeed * i * j
		},
	)
}

func addMatrices(m1, m2 *[][]int) *[][]int {
	res := makeSquareMatrix(matrixSize)
	forEachMatrixElementIndex(
		func(i, j int) { (*res)[i][j] = (*m1)[i][j] + (*m2)[i][j] },
	)
	return res
}

func timeFunction(f func(), iterations int) int64 {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		f()
	}
	elapsed := time.Since(start)

	return timePrecision(elapsed) / int64(iterations)
}

func runOnce() {
	fmt.Println("running once")
	fmt.Println("creating matrices")

	m1 := makeSquareMatrix(matrixSize)
	m2 := makeSquareMatrix(matrixSize)
	fillMatrix(m1)
	fillMatrix(m2)

	var res *[][]int
	time := timeFunction(func() { res = addMatrices(m1, m2) }, 1)

	fmt.Printf("Elapsed %d%s\n", time, timeSuffix)

	if printMatrices {
		fmt.Println("m1:   ", m1)
		fmt.Println("m2:   ", m2)
		fmt.Println("m1+m2:", res)
	}
}

type Entry struct {
	Size int
	Time int
}

type StatResult struct {
	Command string
	Entries []Entry
}

func stat() {
	fmt.Println("running in stat mode")
	fmt.Printf("statting for matrix sizes from %d to %d with a step size of %d\n",
		statStart,
		statEnd,
		statStep,
	)
	fmt.Printf("will be doing %d iterations per matrix size\n", statIterationsPerStep)

	if len(outputPath) == 0 {
		outputPath = "stat_"
		if concurrent {
			outputPath += fmt.Sprintf("threads=%d_", threadCount)
		} else {
			outputPath += "sequential_"
		}
		outputPath += fmt.Sprintf("from=%d_to=%d_step=%d_iterations=%d.json", statStart, statEnd, statStep, statIterationsPerStep)
	}

	fmt.Println("output will be written to", outputPath)

	fmt.Println()

	f, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("couldn't create a file at", outputPath)
		fmt.Println(err)
		os.Exit(1)
	}

	entries := make([]Entry, 0, (statEnd-statStart)/statStep)

	for matrixSize = int(statStart); matrixSize <= int(statEnd); matrixSize += int(statStep) {
		m1 := makeSquareMatrix(matrixSize)
		m2 := makeSquareMatrix(matrixSize)
		fillMatrix(m1)
		fillMatrix(m2)

		time := timeFunction(
			func() { addMatrices(m1, m2) },
			int(statIterationsPerStep),
		)

		entries = append(entries, Entry{
			Size: matrixSize,
			Time: int(time),
		})

		fmt.Printf("statted at %d;elapsed %d%s\n", matrixSize, time, timeSuffix)
	}

	command := strings.Join(os.Args, " ")
	statResult := StatResult{
		Command: command,
		Entries: entries,
	}

	bytes, err := json.Marshal(statResult)
	if err != nil {
		fmt.Println("error converting StatResult to json:")
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = f.Write(bytes)
	if err != nil {
		fmt.Println("error writing to", outputPath)
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	if statConfiguration {
		stat()
	} else {
		runOnce()
	}
}
