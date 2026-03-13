module infini.sh/agent

go 1.24.0

require (
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575
	github.com/pkg/errors v0.9.1
	github.com/shirou/gopsutil/v3 v3.24.5
	github.com/stretchr/testify v1.10.0
	golang.org/x/crypto v0.47.0
	golang.org/x/text v0.34.0
	gopkg.in/yaml.v3 v3.0.1
	infini.sh/framework v0.0.0
)

require (
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/RoaringBitmap/roaring v1.9.4 // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/arl/statsviz v0.6.0 // indirect
	github.com/bits-and-blooms/bitset v1.12.0 // indirect
	github.com/bkaradzic/go-lz4 v1.0.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/caddyserver/certmagic v0.23.0 // indirect
	github.com/caddyserver/zerossl v0.1.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgraph-io/ristretto v0.2.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/gookit/filter v1.2.3 // indirect
	github.com/gookit/goutil v0.7.1 // indirect
	github.com/gookit/validate v1.5.6 // indirect
	github.com/gorilla/context v1.1.2 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/gorilla/sessions v1.4.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/jmoiron/jsonq v0.0.0-20150511023944-e874b168d07e // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kardianos/service v1.2.2 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/libdns/libdns v1.0.0 // indirect
	github.com/libdns/tencentcloud v1.2.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mholt/acmez/v3 v3.1.2 // indirect
	github.com/miekg/dns v1.1.63 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/r3labs/diff/v2 v2.15.1 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/savsgio/gotils v0.0.0-20250408102913-196191ec6287 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.4.1 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zeebo/blake3 v0.2.4 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.uber.org/zap/exp v0.3.0 // indirect
	golang.org/x/mod v0.32.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/term v0.39.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/tools v0.41.0 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace (
	github.com/caddyserver/certmagic => ../vendor/src/github.com/caddyserver/certmagic
	github.com/caddyserver/zerossl => ../vendor/src/github.com/caddyserver/zerossl
	github.com/cihub/seelog => ../vendor/src/github.com/cihub/seelog
	github.com/golang-jwt/jwt => ../vendor/src/github.com/golang-jwt/jwt
	github.com/gopkg.in/gomail.v2 => ../vendor/src/github.com/gopkg.in/gomail.v2
	github.com/libdns/libdns => ../vendor/src/github.com/libdns/libdns
	github.com/libdns/tencentcloud => ../vendor/src/github.com/libdns/tencentcloud
	github.com/quipo/statsd => ../vendor/src/github.com/quipo/statsd
	infini.sh/framework => ../framework
)
