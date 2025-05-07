/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package process

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"infini.sh/framework/core/model"
	"testing"
)

func TestDiscover(t *testing.T) {
	pinfos, err := DiscoverESProcessors(ElasticFilter)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(pinfos)
}

func TestTryGetESClusterInfo(t *testing.T) {
	addr := model.ListenAddr{
		Port: 9206,
		IP:   "*",
	}
	_, info, err := tryGetESClusterInfo(addr)
	fmt.Println(info, err)
}

func TestParsePathValue(t *testing.T) {
	cmdline := "/opt/es/elasticsearch-7.7.1/jdk.app/Contents/Home/bin/java -Xshare:auto -Des.networkaddress.cache.ttl=60 -Des.networkaddress.cache.negative.ttl=10 -XX:+AlwaysPreTouch -Xss1m -Djava.awt.headless=true -Dfile.encoding=UTF-8 -Djna.nosys=true -XX:-OmitStackTraceInFastThrow -XX:+ShowCodeDetailsInExceptionMessages -Dio.netty.noUnsafe=true -Dio.netty.noKeySetOptimization=true -Dio.netty.recycler.maxCapacityPerThread=0 -Dio.netty.allocator.numDirectArenas=0 -Dlog4j.shutdownHookEnabled=false -Dlog4j2.disable.jmx=true -Djava.locale.providers=SPI,COMPAT -Xms1g -Xmx1g -XX:+UseG1GC -XX:G1ReservePercent=25 -XX:InitiatingHeapOccupancyPercent=30 -Djava.io.tmpdir=/var/folders/f6/2gqtmknx4jn357m0vv8151lc0000gn/T/elasticsearch-12464305898562497433 -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=data -XX:ErrorFile=logs/hs_err_pid%p.log -Xlog:gc*,gc+age=trace,safepoint:file=logs/gc.log:utctime,pid,tags:filecount=32,filesize=64m -XX:MaxDirectMemorySize=536870912 -Des.path.home=/opt/es/elasticsearch-7.7.1 -Des.path.conf=/opt/es/elasticsearch-7.7.1/config -Des.distribution.flavor=default -Des.distribution.type=tar -Des.bundled_jdk=true -cp /opt/es/elasticsearch-7.7.1/lib/* org.elasticsearch.bootstrap.Elasticsearch"
	p, _ := parsePathValue(cmdline, `\-Des\.path\.home=(.*?)\s+`)
	fmt.Println(p)
}

func TestElasticFilter(t *testing.T) {
	cmds := []string{
		"/opt/es/elasticsearch-8.3.3/jdk.app/Contents/Home/bin/java -Des.networkaddress.cache.ttl=60 -Des.networkaddress.cache.negative.ttl=10 -Djava.security.manager=allow -XX:+AlwaysPreTouch -Xss1m -Djava.awt.headless=true -Dfile.encoding=UTF-8 -Djna.nosys=true -XX:-OmitStackTraceInFastThrow -Dio.netty.noUnsafe=true -Dio.netty.noKeySetOptimization=true -Dio.netty.recycler.maxCapacityPerThread=0 -Dlog4j.shutdownHookEnabled=false -Dlog4j2.disable.jmx=true -Dlog4j2.formatMsgNoLookups=true -Djava.locale.providers=SPI,COMPAT --add-opens=java.base/java.io=ALL-UNNAMED -XX:+UseG1GC -Djava.io.tmpdir=/var/folders/f6/2gqtmknx4jn357m0vv8151lc0000gn/T/elasticsearch-734978348591728761 -XX:+HeapDumpOnOutOfMemoryError -XX:+ExitOnOutOfMemoryError -XX:HeapDumpPath=data -XX:ErrorFile=logs/hs_err_pid%p.log -Xlog:gc*,gc+age=trace,safepoint:file=logs/gc.log:utctime,pid,tags:filecount=32,filesize=64m -Xms8192m -Xmx8192m -XX:MaxDirectMemorySize=4294967296 -XX:InitiatingHeapOccupancyPercent=30 -XX:G1ReservePercent=25 -Des.distribution.type=tar --module-path /opt/es/elasticsearch-8.3.3/lib -m org.elasticsearch.server/org.elasticsearch.bootstrap.Elasticsearch",
		"/opt/opensearch/opensearch-1.0.0/jdk/bin/java -Xshare:auto -Dopensearch.networkaddress.cache.ttl=60 -Dopensearch.networkaddress.cache.negative.ttl=10 -XX:+AlwaysPreTouch -Xss1m -Djava.awt.headless=true -Dfile.encoding=UTF-8 -Djna.nosys=true -XX:-OmitStackTraceInFastThrow -XX:+ShowCodeDetailsInExceptionMessages -Dio.netty.noUnsafe=true -Dio.netty.noKeySetOptimization=true -Dio.netty.recycler.maxCapacityPerThread=0 -Dio.netty.allocator.numDirectArenas=0 -Dlog4j.shutdownHookEnabled=false -Dlog4j2.disable.jmx=true -Djava.locale.providers=SPI,COMPAT -Xms1g -Xmx1g -XX:+UseG1GC -XX:G1ReservePercent=25 -XX:InitiatingHeapOccupancyPercent=30 -Djava.io.tmpdir=/tmp/opensearch-2153174206831327614 -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=data -XX:ErrorFile=logs/hs_err_pid%p.log -Xlog:gc*,gc+age=trace,safepoint:file=logs/gc.log:utctime,pid,tags:filecount=32,filesize=64m -Dclk.tck=100 -Djdk.attach.allowAttachSelf=true -Djava.security.policy=/opt/opensearch/opensearch-1.0.0/plugins/opensearch-performance-analyzer/pa_config/opensearch_security.policy -XX:MaxDirectMemorySize=536870912 -Dopensearch.path.home=/opt/opensearch/opensearch-1.0.0 -Dopensearch.path.conf=/opt/opensearch/opensearch-1.0.0/config -Dopensearch.distribution.type=tar -Dopensearch.bundled_jdk=true -cp /opt/opensearch/opensearch-1.0.0/lib/* org.opensearch.bootstrap.OpenSearch -d",
		"/opt/search/packages/jdk/15.0.1//bin/java -Xshare:auto -Des.networkaddress.cache.ttl=60 -Des.networkaddress.cache.negative.ttl=10 -XX:+AlwaysPreTouch -Xss1m -Djava.awt.headless=true -Dfile.encoding=UTF-8 -Djna.nosys=true -XX:-OmitStackTraceInFastThrow -XX:+ShowCodeDetailsInExceptionMessages -Dio.netty.noUnsafe=true -Dio.netty.noKeySetOptimization=true -Dio.netty.recycler.maxCapacityPerThread=0 -Dio.netty.allocator.numDirectArenas=0 -Dlog4j.shutdownHookEnabled=false -Dlog4j2.disable.jmx=true -Djava.locale.providers=SPI,COMPAT -Xms1g -Xmx1g -XX:+UseG1GC -XX:G1ReservePercent=25 -XX:InitiatingHeapOccupancyPercent=30 -Djava.io.tmpdir=/tmp/easysearch-1966601411600284833 -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=data -XX:ErrorFile=logs/hs_err_pid%p.log -Xlog:gc*,gc+age=trace,safepoint:file=logs/gc.log:utctime,pid,tags:filecount=32,filesize=64m -XX:MaxDirectMemorySize=536870912 -Des.path.home=/opt/search/instances/easysearch-3node/ccr_node/easysearch-1.1.2-SNAPSHOT -Des.path.conf=/opt/search/instances/easysearch-3node/ccr_node/easysearch-1.1.2-SNAPSHOT/config -Des.distribution.flavor=oss -Des.distribution.type=tar -Des.bundled_jdk=false -cp /opt/search/instances/easysearch-3node/ccr_node/easysearch-1.1.2-SNAPSHOT/lib/* org.easysearch.bootstrap.Easysearch -d",
	}
	for _, cmd := range cmds {
		assert.Equal(t, true, ElasticFilter(cmd))
	}
}
