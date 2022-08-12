package manage

import (
	"fmt"
	"testing"
	"time"
)

func TestGetHostInfoFromConsole(t *testing.T) {
	//fmt.Println(GetHostInfoFromConsole("cbqt35dath2e5f3gtme0"))
	//url := "http://192.168.3.4:8000/agent/instance/cbqt35dath2e5f3gtme0"
	//var req = util.NewGetRequest(url, []byte(""))
	//result, err := util.ExecuteRequest(req)
	//if err != nil {
	//	log.Printf("manage.UploadNodeInfos: uploadNodeInfos failed: %v\n", err)
	//}
	//log.Printf("manage.UploadNodeInfos: upNodeInfo resp: %s\n", string(result.Body))
	//fmt.Println(string(result.Body))

	go chanTest()
	fmt.Printf("waitting\n")
	time.Sleep(time.Second * 30)
	fmt.Printf("complete\n")
}

func chanTest() {
	flagC := make(chan bool)
	go doSomethings(flagC)
	select {
	case <-flagC:
	case <-time.After(time.Second * 5):
		fmt.Println("time out")
	}
	fmt.Println("end")
	close(flagC)
}

func doSomethings(flag chan bool) {
	time.Sleep(time.Second * 10)
	flag <- true
}
