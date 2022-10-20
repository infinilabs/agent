package logs

type LogsModule struct {
}

func (m LogsModule) Setup() {

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
