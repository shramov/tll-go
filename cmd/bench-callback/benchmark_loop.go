// +build !go1.24

package main

type benchState struct {
	i int
	n int
}

var state benchState

// Naive backport of testing.B.Loop function
func (b *B) Loop() bool {
	if state.i < state.n {
		state.i++
		return true
	}

	if state.n != b.N {
		state = benchState{i: 1, n: b.N}
		b.ResetTimer()
		return true
	}
	return false
}
