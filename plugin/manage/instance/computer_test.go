package instance

import (
	"fmt"
	"testing"
	"time"
)

func TestHardwareInfo(t *testing.T) {
	//CollectHostInfo()
	fmt.Println(GetHostInfo())
	fmt.Println(time.Unix(1660824818, 0))
}
