// Authored and revised by YOC team, 2016-2018
// License placeholder #1

// Command bzzhash computes a swarm tree hash.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Yocoin15/Yocoin_Sources/cmd/utils"
	"github.com/Yocoin15/Yocoin_Sources/swarm/storage"
	"gopkg.in/urfave/cli.v1"
)

func hash(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		utils.Fatalf("Usage: swarm hash <file name>")
	}
	f, err := os.Open(args[0])
	if err != nil {
		utils.Fatalf("Error opening file " + args[1])
	}
	defer f.Close()

	stat, _ := f.Stat()
	fileStore := storage.NewFileStore(storage.NewMapChunkStore(), storage.NewFileStoreParams())
	addr, _, err := fileStore.Store(context.TODO(), f, stat.Size(), false)
	if err != nil {
		utils.Fatalf("%v\n", err)
	} else {
		fmt.Printf("%v\n", addr)
	}
}
