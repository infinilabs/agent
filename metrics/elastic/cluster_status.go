package elastic

import (
	"infini.sh/framework/core/elastic"
)

type Metric struct {
	Interfaces []string `config:"interfaces"`
}

func New() (*Metric, error) {
	//cfg := struct {
	//}{}
	//
	//_, err := env.ParseConfig("network", &cfg)
	//if err != nil {
	//	return nil, err
	//}

	me := &Metric{
	}

	return me, nil
}

func (m *Metric) Collect() error {

	client := elastic.GetClient(k)
	stats := client.GetClusterStats()



	return nil
}
