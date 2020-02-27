module github.com/Yocoin15/Yocoin_Sources

go 1.13

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/aristanetworks/goarista v0.0.0-20191023202215-f096da5361bb
	github.com/davecgh/go-spew v1.1.1
	github.com/deckarep/golang-set v1.7.1
	github.com/edsrzf/mmap-go v1.0.0
	github.com/elastic/gosigar v0.10.5
	github.com/fatih/color v1.7.0
	github.com/fjl/memsize v0.0.0-20190710130421-bcb5799ab5e5
	github.com/gizak/termui v0.0.0-00010101000000-000000000000
	github.com/go-stack/stack v1.8.0
	github.com/golang/protobuf v1.3.2
	github.com/golang/snappy v0.0.1
	github.com/hashicorp/golang-lru v0.5.3
	github.com/huin/goupnp v1.0.0
	github.com/influxdata/influxdb v1.7.9
	github.com/jackpal/go-nat-pmp v1.0.1
	github.com/karalabe/hid v1.0.0
	github.com/maruel/panicparse/stack v0.0.0-00010101000000-000000000000 // indirect
	github.com/mattn/go-colorable v0.1.4
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/naoina/toml v0.1.1
	github.com/nsf/termbox-go v0.0.0-20190817171036-93860e161317 // indirect
	github.com/pborman/uuid v1.2.0
	github.com/peterh/liner v1.1.0
	github.com/prometheus/common v0.6.0
	github.com/prometheus/prometheus v0.0.0
	github.com/prometheus/prometheus/util/flock v0.0.0-00010101000000-000000000000
	github.com/rjeczalik/notify v0.9.2
	github.com/robertkrimen/otto v0.0.0-20180617131154-15f95af6e78d
	github.com/rs/cors v1.7.0
	github.com/syndtr/goleveldb v1.0.0
	golang.org/x/crypto v0.0.0-20191112222119-e1110fd1c708
	golang.org/x/net v0.0.0-20191112182307-2180aed22343
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20190912141932-bc967efca4b8
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951
	gopkg.in/olebedev/go-duktape.v3 v3.0.0-20190709231704-1e4459ed25ff
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	gopkg.in/urfave/cli.v1 v1.20.0
)

replace github.com/gizak/termui => ./vendor/github.com/gizak/termui

replace github.com/prometheus/prometheus => ./vendor/github.com/prometheus/prometheus

replace github.com/prometheus/prometheus/util/flock => ./vendor/github.com/prometheus/prometheus/util/flock

replace github.com/maruel/panicparse/stack => ./vendor/github.com/maruel/panicparse/stack

replace github.com/maruel/panicparse => ./vendor/github.com/maruel/panicparse
