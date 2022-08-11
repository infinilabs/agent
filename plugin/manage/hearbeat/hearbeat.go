package hearbeat

import (
	"fmt"
	"infini.sh/agent/api"
	"infini.sh/agent/config"
	"infini.sh/framework/core/errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	TimeOut   time.Duration
	Frequency time.Duration
	Url       string
}

type HeartBeatReqFun func() string
type HeartBeatRespFun func(content string) bool

/*
default client: send heartbeat package to console
*/
func NewDefaultClient(frequency time.Duration, agentId string) Client {
	reqUrl := strings.ReplaceAll(api.UrlHearBeat, ":instance_id", agentId)
	hbUrl := fmt.Sprintf("http://%s:%d/%s", config.EnvConfig.ConsoleConfig.Host, config.EnvConfig.ConsoleConfig.Port, reqUrl)
	return Client{
		TimeOut:   time.Millisecond * 1000,
		Frequency: frequency,
		Url:       hbUrl,
	}
}

func NewSpecClient(frequency time.Duration, timeOut time.Duration, url string) Client {
	return Client{
		TimeOut:   timeOut,
		Frequency: frequency,
		Url:       url,
	}
}

func (c *Client) Heartbeat(reqFuc HeartBeatReqFun, respFunc HeartBeatRespFun) error {
	ticker := time.NewTicker(c.Frequency)
	defer ticker.Stop()

	errCount := 0
	for range ticker.C {
		pck := reqFuc()

		clt := http.Client{
			Timeout: c.TimeOut,
		}
		if errCount > 5 {
			if errCount > 35 { //超过35个周期，则继续监测心跳。针对网络异常的处理，如果是其他错误，则直接停止心跳
				fmt.Println("超过35个周期，继续心跳")
				errCount = 0
			}
			continue
		}
		resp, err := clt.Post(c.Url, "application/json", strings.NewReader(pck))
		if err != nil {
			fmt.Printf("heart beat api error: %v\n", err)
			errCount++
			continue
		}
		bodyContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "send heartbeat content failed")
		}
		respFunc(string(bodyContent))
		//if ok {
		//	log.Printf("heartbeat success: %s", pck)
		//} else {
		//	log.Printf("heartbeat failed: %s", pck)
		//}
		resp.Body.Close()
	}
	return nil
}
