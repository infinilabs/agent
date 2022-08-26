package logs

import "infini.sh/framework/core/config"

type LogsModule struct {
}

func (m LogsModule) Setup(config *config.Config) {

}

func (m LogsModule) Start() error {

	return nil
}

func (m LogsModule) Stop() error {

	return nil
}

func (m LogsModule) Name() string {
	return "logs"
}
