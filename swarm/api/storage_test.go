// Authored and revised by YOC team, 2016-2018
// License placeholder #1

package api

import (
	"context"
	"testing"
)

func testStorage(t *testing.T, f func(*Storage, bool)) {
	testAPI(t, func(api *API, toEncrypt bool) {
		f(NewStorage(api), toEncrypt)
	})
}

func TestStoragePutGet(t *testing.T) {
	testStorage(t, func(api *Storage, toEncrypt bool) {
		content := "hello"
		exp := expResponse(content, "text/plain", 0)
		// exp := expResponse([]byte(content), "text/plain", 0)
		ctx := context.TODO()
		bzzkey, wait, err := api.Put(ctx, content, exp.MimeType, toEncrypt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = wait(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		bzzhash := bzzkey.Hex()
		// to check put against the API#Get
		resp0 := testGet(t, api.api, bzzhash, "")
		checkResponse(t, resp0, exp)

		// check storage#Get
		resp, err := api.Get(context.TODO(), bzzhash)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		checkResponse(t, &testResponse{nil, resp}, exp)
	})
}
