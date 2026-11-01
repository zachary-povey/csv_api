//go:build race

package parser

// the race detector adds allocations, so allocation counts aren't checked
// under go test -race
const raceEnabled = true
