.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/Yocoin15/Yocoin_Sources/cmd/swarm
	go build -o bin/geth     github.com/Yocoin15/Yocoin_Sources/cmd/geth
	go build -o bin/bootnode github.com/Yocoin15/Yocoin_Sources/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/Yocoin15/Yocoin_Sources/swarm/...
