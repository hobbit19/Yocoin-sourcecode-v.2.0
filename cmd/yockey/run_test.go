// Authored and revised by YOC team, 2018
// License placeholder #1

package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/Yocoin15/Yocoin_Sources/internal/cmdtest"
	"github.com/docker/docker/pkg/reexec"
)

type testYockey struct {
	*cmdtest.TestCmd
}

// spawns yockey with the given command line args.
func runYockey(t *testing.T, args ...string) *testYockey {
	tt := new(testYockey)
	tt.TestCmd = cmdtest.NewTestCmd(t, tt)
	tt.Run("ethkey-test", args...)
	return tt
}

func TestMain(m *testing.M) {
	// Run the app if we've been exec'd as "yockey-test" in runYockey.
	reexec.Register("ethkey-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}
