package main

import (
	"flag"
	"fmt"
	"parallel-computations-4/async"
	"parallel-computations-4/utils"
	"sort"
)

// [X] Створити 3 масиви (або колекції) з випадковими числами.
// - [X] У першому масиві - елементи помножити на 5.
// - [X] У другому - залишити тільки парні.
// - [X] У третьому - залишити елементи в діапазоні від 0.4 до 0.6 максимального значення.
// [X] Відсортувати масиви
// [X] і злити в один масив елементи відсортований масив в якому є елементи які входять в усі масиви.

func main() {
	size := flag.Int("size", 10, "size of the arrays")
	flag.Parse()

	arr1Promise := ProcessArrAsync(
		utils.GenerateArrayAsync(*size),
		ProcessArr1,
		"1")
	arr2Promise := ProcessArrAsync(
		utils.GenerateArrayAsync(*size),
		ProcessArr2,
		"2")
	arr3Promise := ProcessArrAsync(
		utils.GenerateArrayAsync(*size),
		ProcessArr3,
		"3")

	arr1 := arr1Promise.Await().([]int)
	arr2 := arr2Promise.Await().([]int)
	arr3 := arr3Promise.Await().([]int)

	outArr := make([]int, 0, len(arr1)+len(arr2)+len(arr3))
	outArr = append(outArr, arr1...)
	outArr = append(outArr, arr2...)
	outArr = append(outArr, arr3...)

	fmt.Println("result:", outArr)
}

func ProcessArrAsync(arrPromise async.Promise, f func([]int) []int, arrNum string) async.Promise {
	return async.RunAsync(func() interface{} {
		var arr []int = arrPromise.Await().([]int)
		fmt.Println("arr          ", arrNum, arr)
		arr = f(arr)
		fmt.Println("processed arr", arrNum, arr)
		sort.Ints(arr)
		fmt.Println("sorted arr   ", arrNum, arr)
		return arr
	})
}

func ProcessArr1(arr []int) []int {
	arrOut := make([]int, 0, len(arr))
	for _, v := range arr {
		arrOut = append(arrOut, v*5)
	}

	return arrOut
}

func ProcessArr2(arr []int) []int {
	arrOut := make([]int, 0, len(arr)/2)
	for _, v := range arr {
		if v%2 == 0 {
			arrOut = append(arrOut, v)
		}
	}

	return arrOut
}

func ProcessArr3(arr []int) []int {
	arrOut := make([]int, 0, len(arr)/5)
	if len(arr) == 0 {
		return arrOut
	}

	max := arr[0]
	for _, v := range arr {
		if v > max {
			max = v
		}
	}

	minBound := int(float32(max) * 0.4)
	maxBound := int(float32(max) * 0.6)

	for _, v := range arr {
		if v >= minBound && v <= maxBound {
			arrOut = append(arrOut, v)
		}
	}

	return arrOut
}
