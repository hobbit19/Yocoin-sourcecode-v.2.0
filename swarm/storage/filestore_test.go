// Authored and revised by YOC team, 2016-2018
// License placeholder #1

package storage

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

const testDataSize = 0x1000000

func TestFileStorerandom(t *testing.T) {
	testFileStoreRandom(false, t)
	testFileStoreRandom(true, t)
}

func testFileStoreRandom(toEncrypt bool, t *testing.T) {
	tdb, cleanup, err := newTestDbStore(false, false)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	db := tdb.LDBStore
	db.setCapacity(50000)
	memStore := NewMemStore(NewDefaultStoreParams(), db)
	localStore := &LocalStore{
		memStore: memStore,
		DbStore:  db,
	}

	fileStore := NewFileStore(localStore, NewFileStoreParams())
	defer os.RemoveAll("/tmp/bzz")

	reader, slice := generateRandomData(testDataSize)
	ctx := context.TODO()
	key, wait, err := fileStore.Store(ctx, reader, testDataSize, toEncrypt)
	if err != nil {
		t.Errorf("Store error: %v", err)
	}
	err = wait(ctx)
	if err != nil {
		t.Fatalf("Store waitt error: %v", err.Error())
	}
	resultReader, isEncrypted := fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	resultSlice := make([]byte, len(slice))
	n, err := resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Errorf("Retrieve error: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error.")
	}
	ioutil.WriteFile("/tmp/slice.bzz.16M", slice, 0666)
	ioutil.WriteFile("/tmp/result.bzz.16M", resultSlice, 0666)
	localStore.memStore = NewMemStore(NewDefaultStoreParams(), db)
	resultReader, isEncrypted = fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	for i := range resultSlice {
		resultSlice[i] = 0
	}
	n, err = resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Errorf("Retrieve error after removing memStore: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error after removing memStore got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error after removing memStore.")
	}
}

func TestFileStoreCapacity(t *testing.T) {
	testFileStoreCapacity(false, t)
	testFileStoreCapacity(true, t)
}

func testFileStoreCapacity(toEncrypt bool, t *testing.T) {
	tdb, cleanup, err := newTestDbStore(false, false)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	db := tdb.LDBStore
	memStore := NewMemStore(NewDefaultStoreParams(), db)
	localStore := &LocalStore{
		memStore: memStore,
		DbStore:  db,
	}
	fileStore := NewFileStore(localStore, NewFileStoreParams())
	reader, slice := generateRandomData(testDataSize)
	ctx := context.TODO()
	key, wait, err := fileStore.Store(ctx, reader, testDataSize, toEncrypt)
	if err != nil {
		t.Errorf("Store error: %v", err)
	}
	err = wait(ctx)
	if err != nil {
		t.Errorf("Store error: %v", err)
	}
	resultReader, isEncrypted := fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	resultSlice := make([]byte, len(slice))
	n, err := resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Errorf("Retrieve error: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error.")
	}
	// Clear memStore
	memStore.setCapacity(0)
	// check whether it is, indeed, empty
	fileStore.ChunkStore = memStore
	resultReader, isEncrypted = fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	if _, err = resultReader.ReadAt(resultSlice, 0); err == nil {
		t.Errorf("Was able to read %d bytes from an empty memStore.", len(slice))
	}
	// check how it works with localStore
	fileStore.ChunkStore = localStore
	//	localStore.dbStore.setCapacity(0)
	resultReader, isEncrypted = fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	for i := range resultSlice {
		resultSlice[i] = 0
	}
	n, err = resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Errorf("Retrieve error after clearing memStore: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error after clearing memStore got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error after clearing memStore.")
	}
}
