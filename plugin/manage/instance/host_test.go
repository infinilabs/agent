package instance

import (
	"fmt"
	"infini.sh/agent/model"
	"infini.sh/framework/core/util"
	"runtime"
	"testing"
)

func BenchmarkGetHostInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		host := &model.Instance{}
		host.IPs = util.GetLocalIPs()
		processInfos := getProcessInfo()
		pathPorts := getNodeConfigPaths(processInfos)
		getClusterConfigs(pathPorts)
	}
}

func BenchmarkRegisterHost(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RegisterInstance()
	}
}

func TestHost(t *testing.T) {
	t.Helper()
	GetInstanceInfo()
	printMem(t)
}

func printMem(t *testing.T) {
	t.Helper()
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	t.Logf("%.2f MB", float64(rtm.Alloc)/1024./1024.)
}

func authInfoError() {
	url := fmt.Sprintf("%s://%s:%d/_nodes/_local", "http://localhost", "http", 9300)
	var req = util.NewGetRequest(url, nil)
	//req.SetBasicAuth("elastic1", "OMWXpuJ014kmuFPvcelr")
	result, err := util.ExecuteRequest(req)
	if err != nil {
		fmt.Printf("出错啦： %v", err)
	}
	fmt.Printf(string(result.Body))
}

var mapStr = `{
  "agent_id": "cblp2tlath26no5qria0",
  "clusters": {
    "elasticsearch": {
      "basic_auth": {},
      "cluster_id": "cbllvrtath255k8ho5e0",
      "cluster_uuid": "KuvqunvmTAyEwx_b13eshg"
    }
  }
}
`

var jsonStr string = `
{
  "_nodes": {
    "total": 1,
    "successful": 1,
    "failed": 0
  },
  "cluster_name": "elasticsearch",
  "nodes": {
    "HLWs24PeT2GaPmozduMEnQ": {
      "name": "chengkaideMacBook-Pro.local",
      "transport_address": "127.0.0.1:9300",
      "host": "127.0.0.1",
      "ip": "127.0.0.1",
      "version": "7.17.5",
      "build_flavor": "default",
      "build_type": "tar",
      "build_hash": "8d61b4f7ddf931f219e3745f295ed2bbc50c8e84",
      "total_indexing_buffer": 858993459,
      "roles": [
        "data",
        "data_cold",
        "data_content",
        "data_frozen",
        "data_hot",
        "data_warm",
        "ingest",
        "master",
        "ml",
        "remote_cluster_client",
        "transform"
      ],
      "attributes": {
        "ml.machine_memory": "17179869184",
        "xpack.installed": "true",
        "transform.node": "true",
        "ml.max_open_jobs": "512",
        "ml.max_jvm_size": "8589934592"
      },
      "settings": {
        "cluster": {
          "name": "elasticsearch",
          "election": {
            "strategy": "supports_voting_only"
          }
        },
        "node": {
          "attr": {
            "transform": {
              "node": "true"
            },
            "xpack": {
              "installed": "true"
            },
            "ml": {
              "max_jvm_size": "8589934592",
              "machine_memory": "17179869184",
              "max_open_jobs": "512"
            }
          },
          "name": "chengkaideMacBook-Pro.local"
        },
        "path": {
          "logs": "/Users/chengkai/Documents/workspace/software/es/logs",
          "home": "/Users/chengkai/Documents/workspace/software/es"
        },
        "client": {
          "type": "node"
        },
        "http": {
          "type": "security4",
          "type.default": "netty4"
        },
        "transport": {
          "type": "security4",
          "features": {
            "x-pack": "true"
          },
          "type.default": "netty4"
        },
        "xpack": {
          "security": {
            "enabled": "true"
          }
        }
      },
      "os": {
        "refresh_interval_in_millis": 1000,
        "name": "Mac OS X",
        "pretty_name": "Mac OS X",
        "arch": "aarch64",
        "version": "12.3",
        "available_processors": 8,
        "allocated_processors": 8
      },
      "process": {
        "refresh_interval_in_millis": 1000,
        "id": 26682,
        "mlockall": false
      },
      "jvm": {
        "pid": 26682,
        "version": "18.0.1.1",
        "vm_name": "OpenJDK 64-Bit Server VM",
        "vm_version": "18.0.1.1+2-6",
        "vm_vendor": "Oracle Corporation",
        "bundled_jdk": true,
        "using_bundled_jdk": true,
        "start_time_in_millis": 1659432800306,
        "mem": {
          "heap_init_in_bytes": 8589934592,
          "heap_max_in_bytes": 8589934592,
          "non_heap_init_in_bytes": 7667712,
          "non_heap_max_in_bytes": 0,
          "direct_max_in_bytes": 0
        },
        "gc_collectors": [
          "G1 Young Generation",
          "G1 Old Generation"
        ],
        "memory_pools": [
          "CodeHeap 'non-nmethods'",
          "Metaspace",
          "CodeHeap 'profiled nmethods'",
          "Compressed Class Space",
          "G1 Eden Space",
          "G1 Old Gen",
          "G1 Survivor Space",
          "CodeHeap 'non-profiled nmethods'"
        ],
        "using_compressed_ordinary_object_pointers": "true",
        "input_arguments": [
          "-Xshare:auto",
          "-Des.networkaddress.cache.ttl=60",
          "-Des.networkaddress.cache.negative.ttl=10",
          "-XX:+AlwaysPreTouch",
          "-Xss1m",
          "-Djava.awt.headless=true",
          "-Dfile.encoding=UTF-8",
          "-Djna.nosys=true",
          "-XX:-OmitStackTraceInFastThrow",
          "-XX:+ShowCodeDetailsInExceptionMessages",
          "-Dio.netty.noUnsafe=true",
          "-Dio.netty.noKeySetOptimization=true",
          "-Dio.netty.recycler.maxCapacityPerThread=0",
          "-Dio.netty.allocator.numDirectArenas=0",
          "-Dlog4j.shutdownHookEnabled=false",
          "-Dlog4j2.disable.jmx=true",
          "-Dlog4j2.formatMsgNoLookups=true",
          "-Djava.locale.providers=SPI,COMPAT",
          "--add-opens=java.base/java.io=ALL-UNNAMED",
          "-Djava.security.manager=allow",
          "-XX:+UseG1GC",
          "-Djava.io.tmpdir=/var/folders/b3/1c4kh8s13m36n2hsbtpn6v2r0000gn/T/elasticsearch-9260088881448917586",
          "-XX:+HeapDumpOnOutOfMemoryError",
          "-XX:+ExitOnOutOfMemoryError",
          "-XX:HeapDumpPath=data",
          "-XX:ErrorFile=logs/hs_err_pid%p.log",
          "-Xlog:gc*,gc+age=trace,safepoint:file=logs/gc.log:utctime,pid,tags:filecount=32,filesize=64m",
          "-Xms8192m",
          "-Xmx8192m",
          "-XX:MaxDirectMemorySize=4294967296",
          "-XX:InitiatingHeapOccupancyPercent=30",
          "-XX:G1ReservePercent=25",
          "-Des.path.home=/Users/chengkai/Documents/workspace/software/es",
          "-Des.path.conf=/Users/chengkai/Documents/workspace/software/es/config",
          "-Des.distribution.flavor=default",
          "-Des.distribution.type=tar",
          "-Des.bundled_jdk=true"
        ]
      },
      "thread_pool": {
        "force_merge": {
          "type": "fixed",
          "size": 1,
          "queue_size": -1
        },
        "search_coordination": {
          "type": "fixed",
          "size": 4,
          "queue_size": 1000
        },
        "ml_datafeed": {
          "type": "scaling",
          "core": 1,
          "max": 512,
          "keep_alive": "1m",
          "queue_size": -1
        },
        "searchable_snapshots_cache_fetch_async": {
          "type": "scaling",
          "core": 0,
          "max": 24,
          "keep_alive": "30s",
          "queue_size": -1
        },
        "snapshot_meta": {
          "type": "scaling",
          "core": 1,
          "max": 24,
          "keep_alive": "30s",
          "queue_size": -1
        },
        "fetch_shard_started": {
          "type": "scaling",
          "core": 1,
          "max": 16,
          "keep_alive": "5m",
          "queue_size": -1
        },
        "listener": {
          "type": "fixed",
          "size": 4,
          "queue_size": -1
        },
        "rollup_indexing": {
          "type": "fixed",
          "size": 1,
          "queue_size": -1
        },
        "search": {
          "type": "fixed_auto_queue_size",
          "size": 13,
          "queue_size": 1000
        },
        "security-crypto": {
          "type": "fixed",
          "size": 4,
          "queue_size": 1000
        },
        "ccr": {
          "type": "fixed",
          "size": 32,
          "queue_size": 100
        },
        "flush": {
          "type": "scaling",
          "core": 1,
          "max": 4,
          "keep_alive": "5m",
          "queue_size": -1
        },
        "fetch_shard_store": {
          "type": "scaling",
          "core": 1,
          "max": 16,
          "keep_alive": "5m",
          "queue_size": -1
        },
        "ml_utility": {
          "type": "scaling",
          "core": 1,
          "max": 2048,
          "keep_alive": "10m",
          "queue_size": -1
        },
        "get": {
          "type": "fixed",
          "size": 8,
          "queue_size": 1000
        },
        "system_read": {
          "type": "fixed",
          "size": 4,
          "queue_size": 2000
        },
        "system_critical_read": {
          "type": "fixed",
          "size": 4,
          "queue_size": 2000
        },
        "write": {
          "type": "fixed",
          "size": 8,
          "queue_size": 10000
        },
        "watcher": {
          "type": "fixed",
          "size": 40,
          "queue_size": 1000
        },
        "security-token-key": {
          "type": "fixed",
          "size": 1,
          "queue_size": 1000
        },
        "system_critical_write": {
          "type": "fixed",
          "size": 4,
          "queue_size": 1500
        },
        "refresh": {
          "type": "scaling",
          "core": 1,
          "max": 4,
          "keep_alive": "5m",
          "queue_size": -1
        },
        "vector_tile_generation": {
          "type": "fixed",
          "size": 1,
          "queue_size": -1
        },
        "system_write": {
          "type": "fixed",
          "size": 4,
          "queue_size": 1000
        },
        "generic": {
          "type": "scaling",
          "core": 4,
          "max": 128,
          "keep_alive": "30s",
          "queue_size": -1
        },
        "warmer": {
          "type": "scaling",
          "core": 1,
          "max": 4,
          "keep_alive": "5m",
          "queue_size": -1
        },
        "auto_complete": {
          "type": "fixed",
          "size": 2,
          "queue_size": 100
        },
        "management": {
          "type": "scaling",
          "core": 1,
          "max": 5,
          "keep_alive": "5m",
          "queue_size": -1
        },
        "analyze": {
          "type": "fixed",
          "size": 1,
          "queue_size": 16
        },
        "searchable_snapshots_cache_prewarming": {
          "type": "scaling",
          "core": 0,
          "max": 16,
          "keep_alive": "30s",
          "queue_size": -1
        },
        "ml_job_comms": {
          "type": "scaling",
          "core": 4,
          "max": 2048,
          "keep_alive": "1m",
          "queue_size": -1
        },
        "snapshot": {
          "type": "scaling",
          "core": 1,
          "max": 4,
          "keep_alive": "5m",
          "queue_size": -1
        },
        "search_throttled": {
          "type": "fixed_auto_queue_size",
          "size": 1,
          "queue_size": 100
        }
      },
      "transport": {
        "bound_address": [
          "[::1]:9300",
          "127.0.0.1:9300"
        ],
        "publish_address": "127.0.0.1:9300",
        "profiles": {}
      },
      "http": {
        "bound_address": [
          "[::1]:9200",
          "127.0.0.1:9200"
        ],
        "publish_address": "127.0.0.1:9200",
        "max_content_length_in_bytes": 104857600
      },
      "plugins": [],
      "modules": [
        {
          "name": "aggs-matrix-stats",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Adds aggregations whose input are a list of numeric fields and output includes a matrix.",
          "classname": "org.elasticsearch.search.aggregations.matrix.MatrixAggregationPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "analysis-common",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Adds \"built in\" analyzers to Elasticsearch.",
          "classname": "org.elasticsearch.analysis.common.CommonAnalysisPlugin",
          "extended_plugins": [
            "lang-painless"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "constant-keyword",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Module for the constant-keyword field type, which is a specialization of keyword for the case when all documents have the same value.",
          "classname": "org.elasticsearch.xpack.constantkeyword.ConstantKeywordMapperPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "frozen-indices",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A plugin for the frozen indices functionality",
          "classname": "org.elasticsearch.xpack.frozen.FrozenIndices",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "ingest-common",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Module for ingest processors that do not require additional security permissions or have large dependencies and resources",
          "classname": "org.elasticsearch.ingest.common.IngestCommonPlugin",
          "extended_plugins": [
            "lang-painless"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "ingest-geoip",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Ingest processor that uses looksup geo data based on ip adresses using the Maxmind geo database",
          "classname": "org.elasticsearch.ingest.geoip.IngestGeoIpPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "ingest-user-agent",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Ingest processor that extracts information from a user agent",
          "classname": "org.elasticsearch.ingest.useragent.IngestUserAgentPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "kibana",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Plugin exposing APIs for Kibana system indices",
          "classname": "org.elasticsearch.kibana.KibanaPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "lang-expression",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Lucene expressions integration for Elasticsearch",
          "classname": "org.elasticsearch.script.expression.ExpressionPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "lang-mustache",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Mustache scripting integration for Elasticsearch",
          "classname": "org.elasticsearch.script.mustache.MustachePlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "lang-painless",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "An easy, safe and fast scripting language for Elasticsearch",
          "classname": "org.elasticsearch.painless.PainlessPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "legacy-geo",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Placeholder plugin for geospatial features in ES",
          "classname": "org.elasticsearch.legacygeo.LegacyGeoPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "mapper-extras",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Adds advanced field mappers",
          "classname": "org.elasticsearch.index.mapper.MapperExtrasPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "mapper-version",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A plugin for a field type to store sofware versions",
          "classname": "org.elasticsearch.xpack.versionfield.VersionFieldPlugin",
          "extended_plugins": [
            "x-pack-core",
            "lang-painless"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "parent-join",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "This module adds the support parent-child queries and aggregations",
          "classname": "org.elasticsearch.join.ParentJoinPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "percolator",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Percolator module adds capability to index queries and query these queries by specifying documents",
          "classname": "org.elasticsearch.percolator.PercolatorPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "rank-eval",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "The Rank Eval module adds APIs to evaluate ranking quality.",
          "classname": "org.elasticsearch.index.rankeval.RankEvalPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "reindex",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "The Reindex module adds APIs to reindex from one index to another or update documents in place.",
          "classname": "org.elasticsearch.reindex.ReindexPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "repositories-metering-api",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Repositories metering API",
          "classname": "org.elasticsearch.xpack.repositories.metering.RepositoriesMeteringPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "repository-encrypted",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - client-side encrypted repositories.",
          "classname": "org.elasticsearch.repositories.encrypted.EncryptedRepositoryPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "repository-url",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Module for URL repository",
          "classname": "org.elasticsearch.plugin.repository.url.URLRepositoryPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "runtime-fields-common",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Module for runtime fields features and extensions that have large dependencies",
          "classname": "org.elasticsearch.runtimefields.RuntimeFieldsCommonPlugin",
          "extended_plugins": [
            "lang-painless"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "search-business-rules",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A plugin for applying business rules to search result rankings",
          "classname": "org.elasticsearch.xpack.searchbusinessrules.SearchBusinessRules",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "searchable-snapshots",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A plugin for the searchable snapshots functionality",
          "classname": "org.elasticsearch.xpack.searchablesnapshots.SearchableSnapshots",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "snapshot-repo-test-kit",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A plugin for a test kit for snapshot repositories",
          "classname": "org.elasticsearch.repositories.blobstore.testkit.SnapshotRepositoryTestKit",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "spatial",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A plugin for Basic Spatial features",
          "classname": "org.elasticsearch.xpack.spatial.SpatialPlugin",
          "extended_plugins": [
            "x-pack-core",
            "legacy-geo"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "transform",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A plugin to transform data",
          "classname": "org.elasticsearch.xpack.transform.Transform",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "transport-netty4",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Netty 4 based transport implementation",
          "classname": "org.elasticsearch.transport.Netty4Plugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "unsigned-long",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Module for the unsigned long field type",
          "classname": "org.elasticsearch.xpack.unsignedlong.UnsignedLongMapperPlugin",
          "extended_plugins": [
            "x-pack-core",
            "lang-painless"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "vector-tile",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A plugin for mapbox vector tile features",
          "classname": "org.elasticsearch.xpack.vectortile.VectorTilePlugin",
          "extended_plugins": [
            "spatial"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "vectors",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A plugin for working with vectors",
          "classname": "org.elasticsearch.xpack.vectors.DenseVectorPlugin",
          "extended_plugins": [
            "x-pack-core",
            "lang-painless"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "wildcard",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A plugin for a keyword field type with efficient wildcard search",
          "classname": "org.elasticsearch.xpack.wildcard.Wildcard",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-aggregate-metric",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Module for the aggregate_metric field type, which allows pre-aggregated fields to be stored a single field.",
          "classname": "org.elasticsearch.xpack.aggregatemetric.AggregateMetricMapperPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-analytics",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Analytics",
          "classname": "org.elasticsearch.xpack.analytics.AnalyticsPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-async",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A module which handles common async operations",
          "classname": "org.elasticsearch.xpack.async.AsyncResultsIndexPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-async-search",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "A module which allows to track the progress of a search asynchronously.",
          "classname": "org.elasticsearch.xpack.search.AsyncSearch",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-autoscaling",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Autoscaling",
          "classname": "org.elasticsearch.xpack.autoscaling.Autoscaling",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-ccr",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - CCR",
          "classname": "org.elasticsearch.xpack.ccr.Ccr",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-core",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Core",
          "classname": "org.elasticsearch.xpack.core.XPackPlugin",
          "extended_plugins": [],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-data-streams",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Data Streams",
          "classname": "org.elasticsearch.xpack.datastreams.DataStreamsPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-deprecation",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Deprecation",
          "classname": "org.elasticsearch.xpack.deprecation.Deprecation",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-enrich",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Enrich",
          "classname": "org.elasticsearch.xpack.enrich.EnrichPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-eql",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "The Elasticsearch plugin that powers EQL for Elasticsearch",
          "classname": "org.elasticsearch.xpack.eql.plugin.EqlPlugin",
          "extended_plugins": [
            "x-pack-ql",
            "lang-painless"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-fleet",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Plugin exposing APIs for Fleet system indices",
          "classname": "org.elasticsearch.xpack.fleet.Fleet",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-graph",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Graph",
          "classname": "org.elasticsearch.xpack.graph.Graph",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-identity-provider",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Identity Provider",
          "classname": "org.elasticsearch.xpack.idp.IdentityProviderPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-ilm",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Index Lifecycle Management",
          "classname": "org.elasticsearch.xpack.ilm.IndexLifecycle",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-logstash",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Logstash",
          "classname": "org.elasticsearch.xpack.logstash.Logstash",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-ml",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Machine Learning",
          "classname": "org.elasticsearch.xpack.ml.MachineLearning",
          "extended_plugins": [
            "x-pack-autoscaling",
            "lang-painless"
          ],
          "has_native_controller": true,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-monitoring",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Monitoring",
          "classname": "org.elasticsearch.xpack.monitoring.Monitoring",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-ql",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch infrastructure plugin for EQL and SQL for Elasticsearch",
          "classname": "org.elasticsearch.xpack.ql.plugin.QlPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-rollup",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Rollup",
          "classname": "org.elasticsearch.xpack.rollup.Rollup",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-security",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Security",
          "classname": "org.elasticsearch.xpack.security.Security",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-shutdown",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Shutdown",
          "classname": "org.elasticsearch.xpack.shutdown.ShutdownPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-sql",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "The Elasticsearch plugin that powers SQL for Elasticsearch",
          "classname": "org.elasticsearch.xpack.sql.plugin.SqlPlugin",
          "extended_plugins": [
            "x-pack-ql",
            "lang-painless"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-stack",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Stack",
          "classname": "org.elasticsearch.xpack.stack.StackPlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-text-structure",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Text Structure",
          "classname": "org.elasticsearch.xpack.textstructure.TextStructurePlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-voting-only-node",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Voting-only node",
          "classname": "org.elasticsearch.cluster.coordination.votingonly.VotingOnlyNodePlugin",
          "extended_plugins": [
            "x-pack-core"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        },
        {
          "name": "x-pack-watcher",
          "version": "7.17.5",
          "elasticsearch_version": "7.17.5",
          "java_version": "1.8",
          "description": "Elasticsearch Expanded Pack Plugin - Watcher",
          "classname": "org.elasticsearch.xpack.watcher.Watcher",
          "extended_plugins": [
            "x-pack-core",
            "lang-painless"
          ],
          "has_native_controller": false,
          "licensed": false,
          "type": "isolated"
        }
      ],
      "ingest": {
        "processors": [
          {
            "type": "append"
          },
          {
            "type": "bytes"
          },
          {
            "type": "circle"
          },
          {
            "type": "community_id"
          },
          {
            "type": "convert"
          },
          {
            "type": "csv"
          },
          {
            "type": "date"
          },
          {
            "type": "date_index_name"
          },
          {
            "type": "dissect"
          },
          {
            "type": "dot_expander"
          },
          {
            "type": "drop"
          },
          {
            "type": "enrich"
          },
          {
            "type": "fail"
          },
          {
            "type": "fingerprint"
          },
          {
            "type": "foreach"
          },
          {
            "type": "geoip"
          },
          {
            "type": "grok"
          },
          {
            "type": "gsub"
          },
          {
            "type": "html_strip"
          },
          {
            "type": "inference"
          },
          {
            "type": "join"
          },
          {
            "type": "json"
          },
          {
            "type": "kv"
          },
          {
            "type": "lowercase"
          },
          {
            "type": "network_direction"
          },
          {
            "type": "pipeline"
          },
          {
            "type": "registered_domain"
          },
          {
            "type": "remove"
          },
          {
            "type": "rename"
          },
          {
            "type": "script"
          },
          {
            "type": "set"
          },
          {
            "type": "set_security_user"
          },
          {
            "type": "sort"
          },
          {
            "type": "split"
          },
          {
            "type": "trim"
          },
          {
            "type": "uppercase"
          },
          {
            "type": "uri_parts"
          },
          {
            "type": "urldecode"
          },
          {
            "type": "user_agent"
          }
        ]
      },
      "aggregations": {
        "adjacency_matrix": {
          "types": [
            "other"
          ]
        },
        "auto_date_histogram": {
          "types": [
            "boolean",
            "date",
            "numeric"
          ]
        },
        "avg": {
          "types": [
            "aggregate_metric",
            "boolean",
            "date",
            "histogram",
            "numeric"
          ]
        },
        "boxplot": {
          "types": [
            "histogram",
            "numeric"
          ]
        },
        "cardinality": {
          "types": [
            "boolean",
            "date",
            "geopoint",
            "geoshape",
            "ip",
            "keyword",
            "numeric",
            "range"
          ]
        },
        "categorize_text": {
          "types": [
            "other"
          ]
        },
        "children": {
          "types": [
            "other"
          ]
        },
        "composite": {
          "types": [
            "other"
          ]
        },
        "date_histogram": {
          "types": [
            "boolean",
            "date",
            "numeric",
            "range"
          ]
        },
        "date_range": {
          "types": [
            "boolean",
            "date",
            "numeric"
          ]
        },
        "diversified_sampler": {
          "types": [
            "boolean",
            "date",
            "keyword",
            "numeric"
          ]
        },
        "extended_stats": {
          "types": [
            "boolean",
            "date",
            "numeric"
          ]
        },
        "filter": {
          "types": [
            "other"
          ]
        },
        "filters": {
          "types": [
            "other"
          ]
        },
        "geo_bounds": {
          "types": [
            "geopoint",
            "geoshape"
          ]
        },
        "geo_centroid": {
          "types": [
            "geopoint",
            "geoshape"
          ]
        },
        "geo_distance": {
          "types": [
            "geopoint"
          ]
        },
        "geo_line": {
          "types": [
            "geopoint"
          ]
        },
        "geohash_grid": {
          "types": [
            "geopoint",
            "geoshape"
          ]
        },
        "geotile_grid": {
          "types": [
            "geopoint",
            "geoshape"
          ]
        },
        "global": {
          "types": [
            "other"
          ]
        },
        "histogram": {
          "types": [
            "boolean",
            "date",
            "histogram",
            "numeric",
            "range"
          ]
        },
        "ip_range": {
          "types": [
            "ip"
          ]
        },
        "matrix_stats": {
          "types": [
            "other"
          ]
        },
        "max": {
          "types": [
            "aggregate_metric",
            "boolean",
            "date",
            "histogram",
            "numeric"
          ]
        },
        "median_absolute_deviation": {
          "types": [
            "numeric"
          ]
        },
        "min": {
          "types": [
            "aggregate_metric",
            "boolean",
            "date",
            "histogram",
            "numeric"
          ]
        },
        "missing": {
          "types": [
            "boolean",
            "date",
            "geopoint",
            "ip",
            "keyword",
            "numeric",
            "range"
          ]
        },
        "multi_terms": {
          "types": [
            "other"
          ]
        },
        "nested": {
          "types": [
            "other"
          ]
        },
        "parent": {
          "types": [
            "other"
          ]
        },
        "percentile_ranks": {
          "types": [
            "boolean",
            "date",
            "histogram",
            "numeric"
          ]
        },
        "percentiles": {
          "types": [
            "boolean",
            "date",
            "histogram",
            "numeric"
          ]
        },
        "range": {
          "types": [
            "boolean",
            "date",
            "histogram",
            "numeric"
          ]
        },
        "rare_terms": {
          "types": [
            "boolean",
            "date",
            "ip",
            "keyword",
            "numeric"
          ]
        },
        "rate": {
          "types": [
            "histogram",
            "numeric"
          ]
        },
        "reverse_nested": {
          "types": [
            "other"
          ]
        },
        "sampler": {
          "types": [
            "other"
          ]
        },
        "scripted_metric": {
          "types": [
            "other"
          ]
        },
        "significant_terms": {
          "types": [
            "boolean",
            "date",
            "ip",
            "keyword",
            "numeric"
          ]
        },
        "significant_text": {
          "types": [
            "other"
          ]
        },
        "stats": {
          "types": [
            "boolean",
            "date",
            "numeric"
          ]
        },
        "string_stats": {
          "types": [
            "keyword"
          ]
        },
        "sum": {
          "types": [
            "aggregate_metric",
            "boolean",
            "date",
            "histogram",
            "numeric"
          ]
        },
        "t_test": {
          "types": [
            "numeric"
          ]
        },
        "terms": {
          "types": [
            "boolean",
            "date",
            "ip",
            "keyword",
            "numeric"
          ]
        },
        "top_hits": {
          "types": [
            "other"
          ]
        },
        "top_metrics": {
          "types": [
            "other"
          ]
        },
        "value_count": {
          "types": [
            "aggregate_metric",
            "boolean",
            "date",
            "geopoint",
            "geoshape",
            "histogram",
            "ip",
            "keyword",
            "numeric",
            "range"
          ]
        },
        "variable_width_histogram": {
          "types": [
            "numeric"
          ]
        },
        "weighted_avg": {
          "types": [
            "numeric"
          ]
        }
      }
    }
  }
}
`
