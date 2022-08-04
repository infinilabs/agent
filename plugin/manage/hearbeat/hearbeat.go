package hearbeat

import (
	"fmt"
	"infini.sh/agent/api"
	"infini.sh/agent/config"
	"infini.sh/framework/core/errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	TimeOut   time.Duration
	Frequency time.Duration
	Url       string
}

type HeartBeatReqFun func() (string, error)
type HeartBeatRespFun func(content string) (bool, error)

/*
default client: send heartbeat package to console
*/
func NewClient(frequency time.Duration) Client {
	return Client{
		TimeOut:   time.Millisecond * 1000,
		Frequency: frequency,
		Url:       fmt.Sprintf("%s%s", config.EnvConfig.Host, api.UrlHearBeat),
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

	for range ticker.C {
		pck, err := reqFuc()
		if err != nil {
			return errors.Wrap(err, "get heartbeat content failed")
		}

		clt := http.Client{
			Timeout: c.TimeOut,
		}
		resp, err := clt.Post(c.Url, "application/json", strings.NewReader(pck))
		if err != nil {
			return errors.Wrap(err, "send heartbeat content failed")
		}
		bodyContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "send heartbeat content failed")
		}
		ok, err := respFunc(string(bodyContent))
		if err != nil {
			return errors.Wrap(err, "parse heartbeat resp content failed")
		}
		if ok {
			log.Printf("check heartbeat success. package: %s", pck)
		} else {
			log.Printf("check heartbeat failed. package: %s", pck)
		}
		resp.Body.Close()
	}
	return nil
}
