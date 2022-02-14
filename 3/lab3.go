package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

// per split bound is [lower; upper)
func ExecutePerSplit(size int, splits int, f func(int, int)) {
	splitSize := size/splits + 1
	for lo := 0; lo < size; lo += splitSize {
		hi := lo + splitSize
		if hi > size {
			hi = size
		}
		f(lo, hi)
	}
}

func ExecutePerSplitParallel(size int, splits int, f func(int, int)) {
	var wg sync.WaitGroup
	ExecutePerSplit(size, splits, func(lo, hi int) {
		wg.Add(1)
		go func() {
			f(lo, hi)
			wg.Done()
		}()
	})
	wg.Wait()
}

func GenerateArray(size int) *[]int {
	arr := make([]int, size)

	ExecutePerSplitParallel(size, 10, func(lo, hi int) {
		randSource := rand.NewSource(time.Now().UnixNano())
		rand := rand.New(randSource)

		for i := lo; i < hi; i++ {
			arr[i] = rand.Intn(100)
		}
	})

	return &arr
}

func CountDivisibleBy5Sequential(arr *[]int) int {
	numOfDivisibleBy5 := 0
	for _, v := range *arr {
		if v%5 == 0 {
			numOfDivisibleBy5++
		}
	}
	return numOfDivisibleBy5
}

func CountDivisibleBy5ParallelBlocking(arr *[]int) int {
	numOfDivisibleBy5 := 0
	var mutex sync.Mutex

	ExecutePerSplitParallel(len(*arr), 10, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			if (*arr)[i]%5 == 0 {
				mutex.Lock()
				numOfDivisibleBy5++
				mutex.Unlock()
			}
		}
	})

	return numOfDivisibleBy5
}

func CountDivisibleBy5Parallel(arr *[]int) int {
	numOfDivisibleBy5 := 0
	var mutex sync.Mutex

	ExecutePerSplitParallel(len(*arr), 10, func(lo, hi int) {
		threadLocalNumOfDivisibleBy5 := 0
		for i := lo; i < hi; i++ {
			if (*arr)[i]%5 == 0 {
				threadLocalNumOfDivisibleBy5++
			}
		}
		mutex.Lock()
		numOfDivisibleBy5 += threadLocalNumOfDivisibleBy5
		mutex.Unlock()
	})

	return numOfDivisibleBy5
}

func timeFunction(f func(), iterations int) int64 {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		f()
	}
	elapsed := time.Since(start)

	return elapsed.Nanoseconds() / int64(iterations)
}

type AlgorithmToTest struct {
	Name      string
	Algorithm func(*[]int) int
	Enabled   bool
}

func main() {
	seqEnabled := flag.Bool("sequential", false, "run sequential algorithm")
	blockingEnabled := flag.Bool("blocking", false, "run blocking algorithm")
	parallelEnabled := flag.Bool("parallel", false, "run parallel algorithm")

	size := flag.Int("size", 1000, "size of an array")
	iterationsPerAlgorithm := flag.Int("iterations", 1, "number of iterations to do per algorithm when timing it")

	printHelp := flag.Bool("help", false, "print this message")

	flag.Parse()

	if *printHelp {
		flag.Usage()
		os.Exit(0)
	}

	if !(*seqEnabled || *blockingEnabled || *parallelEnabled) {
		*parallelEnabled = true
	}

	algorithms := []AlgorithmToTest{
		{
			"Sequential",
			CountDivisibleBy5Sequential,
			*seqEnabled,
		},
		{
			"Blocking",
			CountDivisibleBy5ParallelBlocking,
			*blockingEnabled,
		},
		{
			"Parallel",
			CountDivisibleBy5Parallel,
			*parallelEnabled,
		},
	}

	fmt.Println("Generating an array...")
	arr := GenerateArray(*size)

	for _, algo := range algorithms {
		if !algo.Enabled {
			continue
		}

		fmt.Printf("Timing %s...\n", algo.Name)
		res := timeFunction(func() { algo.Algorithm(arr) }, *iterationsPerAlgorithm)
		fmt.Printf("Takes %d ns per iteration\n", res)
	}
}
