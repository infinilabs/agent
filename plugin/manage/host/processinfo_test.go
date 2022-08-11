package host

import (
	"fmt"
	"testing"
)

func BenchmarkGetProcessInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getProcessInfo()
	}
}

func BenchmarkGetPort(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPortByPid("22351")
	}
}

func TestProcessInfo(t *testing.T) {
	fmt.Println(getProcessInfo())
}
