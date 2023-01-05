/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package process

import (
	"fmt"
	"infini.sh/agent/model"
	"infini.sh/framework/core/elastic"
	"testing"
)

func TestDiscover(t *testing.T){
	pinfos, err := Discover(ElasticFilter)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(pinfos)
}

func TestDiscoverFromEndpoint(t *testing.T) {
	cfg := elastic.ElasticsearchConfig{
		Endpoint: "http://127.0.0.1:9200",
		Enabled: true,
	}
	cfg.ID = "default"
	nodes, err := DiscoverESNodeFromEndpoint(cfg)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(nodes)
}

func TestTryGetESClusterInfo(t *testing.T)  {
	addr := model.ListenAddr{
		Port: 9206,
		IP: "*",
	}
	_, info, err := tryGetESClusterInfo(addr)
	fmt.Println(info, err)
}

func TestDiscoverESNode(t *testing.T)  {
	info, err := DiscoverESNode(nil)
	fmt.Println(info, err)
}

func TestParsePathValue(t *testing.T) {
	cmdline := "/opt/es/elasticsearch-7.7.1/jdk.app/Contents/Home/bin/java -Xshare:auto -Des.networkaddress.cache.ttl=60 -Des.networkaddress.cache.negative.ttl=10 -XX:+AlwaysPreTouch -Xss1m -Djava.awt.headless=true -Dfile.encoding=UTF-8 -Djna.nosys=true -XX:-OmitStackTraceInFastThrow -XX:+ShowCodeDetailsInExceptionMessages -Dio.netty.noUnsafe=true -Dio.netty.noKeySetOptimization=true -Dio.netty.recycler.maxCapacityPerThread=0 -Dio.netty.allocator.numDirectArenas=0 -Dlog4j.shutdownHookEnabled=false -Dlog4j2.disable.jmx=true -Djava.locale.providers=SPI,COMPAT -Xms1g -Xmx1g -XX:+UseG1GC -XX:G1ReservePercent=25 -XX:InitiatingHeapOccupancyPercent=30 -Djava.io.tmpdir=/var/folders/f6/2gqtmknx4jn357m0vv8151lc0000gn/T/elasticsearch-12464305898562497433 -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=data -XX:ErrorFile=logs/hs_err_pid%p.log -Xlog:gc*,gc+age=trace,safepoint:file=logs/gc.log:utctime,pid,tags:filecount=32,filesize=64m -XX:MaxDirectMemorySize=536870912 -Des.path.home=/opt/es/elasticsearch-7.7.1 -Des.path.conf=/opt/es/elasticsearch-7.7.1/config -Des.distribution.flavor=default -Des.distribution.type=tar -Des.bundled_jdk=true -cp /opt/es/elasticsearch-7.7.1/lib/* org.elasticsearch.bootstrap.Elasticsearch"
	p, _ := parsePathValue(cmdline, `\-Des\.path\.home=(.*?)\s+`)
	fmt.Println(p)
}