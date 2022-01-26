package diagnostics

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
	elastic2 "infini.sh/framework/modules/metrics/elastic"
	"path"
)

type DiagnosticsAnalysisModule struct {
	ClusterID     string `config:"cluster_id"`
	Path          string `config:"path"`
	NodeStats     bool   `config:"node_stats"`
	IndexStats    bool   `config:"index_stats"`
	AllIndexStats bool   `config:"all_index_stats"`
	ShardsStats bool   `config:"shard_stats"`
}

func (module *DiagnosticsAnalysisModule) Name() string {
	return "diagnostics"
}

func (module *DiagnosticsAnalysisModule) Setup(cfg *config.Config) {
	cfg.Unpack(&module)
	fmt.Println(module)
}

func (module *DiagnosticsAnalysisModule) Start() error {

	esMtrics := elastic2.Metric{}
	if module.NodeStats{
		nodes := path.Join(module.Path, "nodes_stats.json")
		bytes, err := util.FileGetContent(nodes)
		if err != nil {
			panic(err)
		}

		obj := elastic.NodesStats{}
		err = util.FromJSONBytes(bytes, &obj)
		if err != nil {
			panic(err)
		}

		esMtrics.SaveNodeStats(module.ClusterID, &obj)
	}

	if module.IndexStats{
		indices := path.Join(module.Path, "indices_stats.json")
		bytes, err := util.FileGetContent(indices)
		if err != nil {
			panic(err)
		}

		obj := &elastic.Stats{}
		err = util.FromJSONBytes(bytes, obj)
		if err != nil {
			panic(err)
		}
		if obj != nil {

			if module.AllIndexStats {
				esMtrics.SaveIndexStats(module.ClusterID, "_all", "_all", obj.All.Primaries, obj.All.Total)
			}

			if module.IndexStats {
				for x, y := range obj.Indices {
					esMtrics.SaveIndexStats(module.ClusterID, y.Uuid, x, y.Primaries, y.Total)
				}
			}
		}
	}

	if module.ShardsStats{
		indices := path.Join(module.Path, "shards.json")
		bytes, err := util.FileGetContent(indices)
		if err != nil {
			panic(err)
		}

		obj := &[]elastic.CatShardResponse{}
		err = util.FromJSONBytes(bytes, obj)
		if err != nil {
			panic(err)
		}
		if obj != nil {

			//  "index" : "contacts_205_v4",
			//	"shard" : "1",
			//	"prirep" : "r",
			//	"state" : "STARTED",
			//	"docs" : "5560550",
			//	"store" : "30874360911",
			//	"ip" : "10.128.2.124",
			//	"node" : "es7-main-124"

			nodeLevelResult:=map[string]NestedTreeMap{}
			for _,v:=range *obj{
				x,ok:=nodeLevelResult[v.NodeIP]
				if !ok{
					x=NestedTreeMap{}
					x.Brand=v.NodeName
					x.Name=v.NodeIP
				}
				docs,err:=util.ToInt(v.Docs)
				if err!=nil{
					panic(err)
				}
				x.Value+=int64(docs)
				x.Children=append(x.Children,NestedTreeMap{Name: v.NodeIP,Value: int64(docs),Payload: v})

				nodeLevelResult[v.NodeIP]=x
			}

			fmt.Println(string(util.MustToJSONBytes(nodeLevelResult)))
		}
	}
	return nil
}
func (module *DiagnosticsAnalysisModule) Stop() error {

	return nil
}

type NestedTreeMap struct {
	Payload elastic.CatShardResponse
	Brand string
	Name string
	Value int64
	Children []NestedTreeMap
}