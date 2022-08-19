package hearbeat

import (
	"fmt"
	log "github.com/cihub/seelog"
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
	hbUrl := fmt.Sprintf("%s/%s", config.GetManagerEndpoint(), reqUrl)
	return Client{
		TimeOut:   time.Millisecond * 1000,
		Frequency: frequency,
		Url:       hbUrl,
	}
}

func NewClient(frequency time.Duration, timeOut time.Duration, url string) Client {
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

		resp, err := clt.Post(c.Url, "application/json", strings.NewReader(pck))
		if err != nil {
			log.Errorf("heart beat api error: %v\n", err)
			errCount++
			continue
		}
		bodyContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "send heartbeat content failed")
		}
		respFunc(string(bodyContent))
		resp.Body.Close()
	}
	return nil
}
