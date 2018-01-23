package main

import (
	"math/rand"
	"time"
)

func getItems() []int {
	x := getRand()
	if x%2 == 0 {
		return []int{1, 2, 3, 4, 5}
	}
	return []int{}
	// return []int{1, 2, 3, 4, 5}
}
func getRand() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(10)
}
