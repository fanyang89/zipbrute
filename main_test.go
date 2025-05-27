package main

import "testing"

func TestEnd(t *testing.T) {
	ok := isEnd([]int{61, 61})
	if !ok {
		t.Fail()
	}
}
