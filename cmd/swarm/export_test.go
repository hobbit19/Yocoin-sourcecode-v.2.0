// Authored and revised by YOC team, 2018
// License placeholder #1

package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Yocoin15/Yocoin_Sources/swarm"
)

// TestCLISwarmExportImport perform the following test:
// 1. runs swarm node
// 2. uploads a random file
// 3. runs an export of the local datastore
// 4. runs a second swarm node
// 5. imports the exported datastore
// 6. fetches the uploaded random file from the second node
func TestCLISwarmExportImport(t *testing.T) {
	cluster := newTestCluster(t, 1)

	// generate random 10mb file
	f, cleanup := generateRandomFile(t, 10000000)
	defer cleanup()

	// upload the file with 'swarm up' and expect a hash
	up := runSwarm(t, "--bzzapi", cluster.Nodes[0].URL, "up", f.Name())
	_, matches := up.ExpectRegexp(`[a-f\d]{64}`)
	up.ExpectExit()
	hash := matches[0]

	var info swarm.Info
	if err := cluster.Nodes[0].Client.Call(&info, "bzz_info"); err != nil {
		t.Fatal(err)
	}

	cluster.Stop()
	defer cluster.Cleanup()

	// generate an export.tar
	exportCmd := runSwarm(t, "db", "export", info.Path+"/chunks", info.Path+"/export.tar", strings.TrimPrefix(info.BzzKey, "0x"))
	exportCmd.ExpectExit()

	// start second cluster
	cluster2 := newTestCluster(t, 1)

	var info2 swarm.Info
	if err := cluster2.Nodes[0].Client.Call(&info2, "bzz_info"); err != nil {
		t.Fatal(err)
	}

	// stop second cluster, so that we close LevelDB
	cluster2.Stop()
	defer cluster2.Cleanup()

	// import the export.tar
	importCmd := runSwarm(t, "db", "import", info2.Path+"/chunks", info.Path+"/export.tar", strings.TrimPrefix(info2.BzzKey, "0x"))
	importCmd.ExpectExit()

	// spin second cluster back up
	cluster2.StartExistingNodes(t, 1, strings.TrimPrefix(info2.BzzAccount, "0x"))

	// try to fetch imported file
	res, err := http.Get(cluster2.Nodes[0].URL + "/bzz:/" + hash)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected HTTP status %d, got %s", 200, res.Status)
	}

	// compare downloaded file with the generated random file
	mustEqualFiles(t, f, res.Body)
}

func mustEqualFiles(t *testing.T, up io.Reader, down io.Reader) {
	h := md5.New()
	upLen, err := io.Copy(h, up)
	if err != nil {
		t.Fatal(err)
	}
	upHash := h.Sum(nil)
	h.Reset()
	downLen, err := io.Copy(h, down)
	if err != nil {
		t.Fatal(err)
	}
	downHash := h.Sum(nil)

	if !bytes.Equal(upHash, downHash) || upLen != downLen {
		t.Fatalf("downloaded imported file md5=%x (length %v) is not the same as the generated one mp5=%x (length %v)", downHash, downLen, upHash, upLen)
	}
}

func generateRandomFile(t *testing.T, size int) (f *os.File, teardown func()) {
	// create a tmp file
	tmp, err := ioutil.TempFile("", "swarm-test")
	if err != nil {
		t.Fatal(err)
	}

	// callback for tmp file cleanup
	teardown = func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}

	// write 10mb random data to file
	buf := make([]byte, 10000000)
	_, err = rand.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	ioutil.WriteFile(tmp.Name(), buf, 0755)

	return tmp, teardown
}
