package elastic

import (
	"fmt"
	"os"
	"testing"
)

func TestESInfo(t *testing.T) {
	t.Skip()
}

func readFile(fname string) {
	file, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	buf := make([]byte, 62)
	stat, err := os.Stat(fname)
	start := stat.Size() - 62
	_, err = file.ReadAt(buf, start)
	if err == nil {
		fmt.Printf("%s\n", buf)
	}

}
