/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package harvester

import (
	"infini.sh/agent/lib/reader"
	"log"
	"testing"
)

func TestReader(t *testing.T) {
	readJsonFile()
}

func readJsonFile() {
	//25101
	var offset int64 = 0
	h, err := NewHarvester("/Users/chengkai/Documents/workspace/software/es/logs/cluster-ck-local-9100_server.json", offset)
	if err != nil {
		log.Println(err)
		return
	}
	r, err := h.NewJsonFileReader("", true)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("start reading>>>>")
	var msg reader.Message
	for i := 0; i < 10; i++ {
		msg, err = r.Next()
		if err != nil {
			log.Println(err)
			break
		}
		log.Println(string(msg.Content))
		log.Printf("offset: %d, size: %d, line: %d\n\n", msg.Offset, msg.Bytes, msg.LineNumbers)
	}
}
