package host

import (
	"fmt"
	"regexp"
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
	//fmt.Println(getProcessInfo())
	content := `"   -Des.distribution.type="zip"   -Des.bundled_jdk="true"   -cp "C:\CI\elasticsearch-7.14.0\lib\*" "org.elasticsearch.bootstrap.Elasticsearch"           java.exe                        3448`
	re := regexp.MustCompile(`java.exe(.*)?([1-9]+)?`)
	result := re.FindAllStringSubmatch(content, -1)
	if result != nil {
		fmt.Println(result[0][1])
	}
}
