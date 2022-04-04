package utils

import (
	"math/rand"
	"parallel-computations-4/async"
	"time"
)

func GenerateArrayAsync(size int) async.Promise {
	return async.RunAsync(func() interface{} {
		return GenerateArray(size)
	})
}

func GenerateArray(size int) []int {
	randSource := rand.NewSource(time.Now().UnixNano())
	rand := rand.New(randSource)

	arr := make([]int, 0, size)
	for i := 0; i < size; i++ {
		arr = append(arr, rand.Intn(100))
	}

	return arr
}
