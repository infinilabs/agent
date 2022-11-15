package diagnostics

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
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

func (module *DiagnosticsAnalysisModule) Setup() {
	//TODO
	//cfg.Unpack(&module)
	//fmt.Println(module)
}

func (module *DiagnosticsAnalysisModule) Start() error {

	//esMtrics := elastic2.Metric{}
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

		//esMtrics.SaveNodeStats(module.ClusterID, &obj)
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
				//esMtrics.SaveIndexStats(module.ClusterID, "_all", "_all", obj.All.Primaries, obj.All.Total)
			}

			if module.IndexStats {
				//for x, y := range obj.Indices {
				//	esMtrics.SaveIndexStats(module.ClusterID, y.Uuid, x, y.Primaries, y.Total)
				//}
			}
		}
	}

	if module.ShardsStats{
		indices := path.Join(module.Path, "shards.json")
		bytes, err := util.FileGetContent(indices)
		if err != nil {
			panic(err)
		}

		//root := NestedTreeMap{Name: "root"}

		obj := &[]elastic.CatShardResponse{}
		err = util.FromJSONBytes(bytes, obj)
		if err != nil {
			panic(err)
		}
		if obj != nil {

			//return module.parseSingleLevel(obj)

			module.indexLevel(obj)
			//module.nodeShardLevel(obj, root)
		}
	}
	return nil
}

func (module *DiagnosticsAnalysisModule) indexLevel(obj *[]elastic.CatShardResponse) {
	var byStore = true
	var match = []string{}

	//  "index" : "contacts_205_v4",
	//	"shard" : "1",
	//	"prirep" : "r",
	//	"state" : "STARTED",
	//	"docs" : "5560550",
	//	"store" : "30874360911",
	//	"ip" : "10.128.2.124",
	//	"node" : "es7-main-124"

	indexLevelResult := map[string][]elastic.CatShardResponse{}
	for _, v := range *obj {
		//check shards
		a, ok := indexLevelResult[v.Index]
		if !ok {
			a = []elastic.CatShardResponse{}
		}
		a = append(a, v)
		indexLevelResult[v.Index] = a
	}

	root:=NestedTreeMap{Name:"root"}

		//each shard
		for indexName1, d := range indexLevelResult {
			var smallShards = 0
			var largeShards = 0

			if len(match) > 0 && !util.ContainsAnyInArray(indexName1, match) {
				continue
			}

			var indexLevelStoreSizeTotal int64 = 0
			var indexLevelDocSizeTotal int64 = 0
			var indexLevelShardSizeTotal int64 = 0
			index := NestedTreeMap{Name: indexName1}

			//each shard
			for _, v := range d {
				shard := NestedTreeMap{Name: fmt.Sprintf("%v", v.ShardID)}

				indexLevelShardSizeTotal++

				if v.Store == "" {
					continue
					v.Store = "0"
				}
				storeSize, err := util.ToInt64(v.Store)
				if err != nil {
					panic(err)
				}
				if v.Docs == "" {
					continue
					v.Docs = "0"
				}
				docSize, err := util.ToInt64(v.Docs)
				if err != nil {
					panic(err)
				}
				indexLevelDocSizeTotal += int64(docSize)
				indexLevelStoreSizeTotal += int64(storeSize)
				if byStore {
					shard.Value = storeSize
				} else {
					shard.Value = docSize
				}

				if storeSize > 10*1024*1024*1024 {
					largeShards++
				} else {
					smallShards++
				}

				shard.Name = fmt.Sprintf("%v[%v] (%v)(%v)", v.ShardType, v.ShardID, util.ByteSize(uint64(storeSize)), util.NearestThousandFormat(float64(docSize)))
				//index.Children = append(index.Children, shard)
		}

		if byStore {
			index.Value = indexLevelStoreSizeTotal
		} else {
			index.Value = indexLevelDocSizeTotal
		}
		////by shars number
		//index.Value=indexLevelShardSizeTotal

		index.Brand=fmt.Sprintf("shards:%v",indexLevelShardSizeTotal)
		index.Name = fmt.Sprintf("%v(%v)(%v)(%v)", indexName1, util.ByteSize(uint64(indexLevelStoreSizeTotal)), util.NearestThousandFormat(float64(indexLevelDocSizeTotal)),indexLevelShardSizeTotal)
		root.Children = append(root.Children, index)
	}

	fmt.Println(string(util.MustToJSONBytes(root)))

	for _, v := range root.Children {
		util.FilePutContentWithByte(path.Join(global.Env().GetDataDir(), v.Name), util.MustToJSONBytes(v))
	}
}

func (module *DiagnosticsAnalysisModule) nodeShardLevel(obj *[]elastic.CatShardResponse, root NestedTreeMap) {
	var byStore = false
	var match = []string{}

	//  "index" : "contacts_205_v4",
	//	"shard" : "1",
	//	"prirep" : "r",
	//	"state" : "STARTED",
	//	"docs" : "5560550",
	//	"store" : "30874360911",
	//	"ip" : "10.128.2.124",
	//	"node" : "es7-main-124"

	nodeLevelResult := map[string]map[string][]elastic.CatShardResponse{}
	for _, v := range *obj {
		//check node
		a, ok := nodeLevelResult[v.NodeIP]
		if !ok {
			a = map[string][]elastic.CatShardResponse{}
		}

		//check index
		b, ok := a[v.Index]
		if !ok {
			b = []elastic.CatShardResponse{}
		}

		b = append(b, v)
		a[v.Index] = b
		nodeLevelResult[v.NodeIP] = a

		//docs,err:=util.ToInt(v.Docs)
		//if err!=nil{
		//	panic(err)
		//}
		//x.Value+=int64(docs)
		//x.Children=append(x.Children,NestedTreeMap{Name: v.NodeIP,Value: int64(docs),Payload: v})
		//
		//nodeLevelResult[v.NodeIP]=x
	}

	//nodeLevelResult:=map[string]map[string][]elastic.CatShardResponse{}

	//var byStore=true
	//var match=[]string{"accounts_","contacts_"}
	//root:=NestedTreeMap{Name:"root"}

	//each node
	for a, b := range nodeLevelResult {
		var nodeLevelStoreSizeTotal int64 = 0
		var nodeLevelDocSizeTotal int64 = 0
		node := NestedTreeMap{Name: a}
		var smallShards = 0
		var largeShards = 0
		//each index
		for c, d := range b {

			if len(match) > 0 && !util.ContainsAnyInArray(c, match) {
				continue
			}

			var indexLevelStoreSizeTotal int64 = 0
			var indexLevelDocSizeTotal int64 = 0
			index := NestedTreeMap{Name: c}

			//each shard
			for _, v := range d {
				shard := NestedTreeMap{Name: fmt.Sprintf("%v", v.ShardID)}
				if v.Store == "" {
					continue
					v.Store = "0"
				}
				storeSize, err := util.ToInt64(v.Store)
				if err != nil {
					panic(err)
				}
				if v.Docs == "" {
					continue
					v.Docs = "0"
				}
				docSize, err := util.ToInt64(v.Docs)
				if err != nil {
					panic(err)
				}
				indexLevelDocSizeTotal += int64(docSize)
				indexLevelStoreSizeTotal += int64(storeSize)
				if byStore {
					shard.Value = storeSize
				} else {
					shard.Value = docSize
				}

				if storeSize > 10*1024*1024*1024 {
					largeShards++
				} else {
					smallShards++
				}

				shard.Name = fmt.Sprintf("%v[%v] (%v)(%v)", v.ShardType, v.ShardID, util.ByteSize(uint64(storeSize)), util.NearestThousandFormat(float64(docSize)))
				index.Children = append(index.Children, shard)
			}
			nodeLevelStoreSizeTotal += indexLevelStoreSizeTotal
			nodeLevelDocSizeTotal += indexLevelDocSizeTotal
			if byStore {
				index.Value = indexLevelStoreSizeTotal
			} else {
				index.Value = indexLevelDocSizeTotal
			}

			index.Name = fmt.Sprintf("%v(%v)(%v)", c, util.ByteSize(uint64(indexLevelStoreSizeTotal)), util.NearestThousandFormat(float64(indexLevelDocSizeTotal)))
			node.Children = append(node.Children, index)
		}

		if byStore {
			node.Value = nodeLevelStoreSizeTotal
		} else {
			node.Value = nodeLevelDocSizeTotal
		}
		node.Brand = fmt.Sprintf("large:%v,small:%v", largeShards, smallShards)
		node.Name = fmt.Sprintf("%v(%v)(%v)", a, util.ByteSize(uint64(nodeLevelStoreSizeTotal)), util.NearestThousandFormat(float64(nodeLevelDocSizeTotal)))
		root.Children = append(root.Children, node)
	}

	fmt.Println(string(util.MustToJSONBytes(root)))
	for _, v := range root.Children {
		util.FilePutContentWithByte(path.Join(global.Env().GetDataDir(), v.Name), util.MustToJSONBytes(v))
	}
}

func (module *DiagnosticsAnalysisModule) parseSingleLevel(obj *[]elastic.CatShardResponse) error {
	//only shard size by index
	var byStore = false
	//var match=[]string{"accounts_","contacts_"}
	root := NestedTreeMap{Name: "root"}
	for _, v := range *obj {
		shard := NestedTreeMap{}
		if v.Store == "" {
			v.Store = "0"
		}
		storeSize, err := util.ToInt64(v.Store)
		if err != nil {
			log.Error(v, err, util.MustToJSON(v))
			panic(err)
		}
		if v.Docs == "" {
			v.Docs = "0"
		}
		docSize, err := util.ToInt64(v.Docs)
		if err != nil {
			panic(err)
		}
		if byStore {
			shard.Value = storeSize
		} else {
			shard.Value = docSize
		}
		shard.Name = fmt.Sprintf("%v[%v] (%v)(%v)", v.ShardType, v.ShardID, util.ByteSize(uint64(storeSize)), util.NearestThousandFormat(float64(docSize)))

		root.Children = append(root.Children, shard)
	}

	fmt.Println(string(util.MustToJSONBytes(root)))

	return nil
}

func (module *DiagnosticsAnalysisModule) Stop() error {

	return nil
}

type NestedTreeMap struct {
	Brand string `json:"brand,omitempty"`
	Name string `json:"name,omitempty"`
	Value int64 `json:"value,omitempty"`
	Children []NestedTreeMap `json:"children,omitempty"`
}
