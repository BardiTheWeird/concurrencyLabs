package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

var (
	matrixSize    int
	printMatrices bool
	concurrent    bool
	threadCount   uint

	statConfiguration bool

	forEachMatrixElementIndex func(func(int, int))

	_rand rand.Rand
)

func init() {
	flag.BoolVar(&printMatrices, "p", false, "whether to print input matrices")
	flag.IntVar(&matrixSize, "size", 3, "size of the input matrices")
	flag.BoolVar(&concurrent, "concurrent", false, "whether to run concurrently")
	flag.UintVar(&threadCount, "threads", 0, "number of threads excluding a producer thread")
	flag.BoolVar(&statConfiguration, "stat", false, "whether to output an elaborate statistic")

	flag.Parse()

	if matrixSize <= 0 {
		fmt.Printf("Matrix size <= 0, exiting...")
		os.Exit(1)
	}

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

	fmt.Println("printMatrices:", printMatrices)
	fmt.Println("matrixSize:", matrixSize)
	fmt.Println("concurrent:", concurrent)
	fmt.Println("threadCount:", threadCount)

	fmt.Println()

	randSource := rand.NewSource(time.Now().UnixNano())
	_rand = *rand.New(randSource)
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

	return elapsed.Milliseconds()
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

	fmt.Printf("Elapsed %dms\n", time)

	if printMatrices {
		fmt.Println("m1:   ", m1)
		fmt.Println("m2:   ", m2)
		fmt.Println("m1+m2:", res)
	}
}

func stat() {

}

func main() {
	if statConfiguration {
		stat()
	} else {
		runOnce()
	}
}
