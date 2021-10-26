/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package store

import (
	"infini.sh/agent/metrics/common"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"time"
)

func Save(metricType string,event common.Event)error {
	event.Timestamp=time.Now()
	log.Error(metricType,",",event.String())

	queue.Push("metrics",util.MustToJSONBytes(event))

	return nil
}
