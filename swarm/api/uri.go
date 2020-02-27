// Authored and revised by YOC team, 2017-2018
// License placeholder #1

package api

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/swarm/storage"
)

//matches hex swarm hashes
// TODO: this is bad, it should not be hardcoded how long is a hash
var hashMatcher = regexp.MustCompile("^([0-9A-Fa-f]{64})([0-9A-Fa-f]{64})?$")

// URI is a reference to content stored in swarm.
type URI struct {
	// Scheme has one of the following values:
	//
	// * bzz           - an entry in a swarm manifest
	// * bzz-raw       - raw swarm content
	// * bzz-immutable - immutable URI of an entry in a swarm manifest
	//                   (address is not resolved)
	// * bzz-list      -  list of all files contained in a swarm manifest
	//
	Scheme string

	// Addr is either a hexadecimal storage address or it an address which
	// resolves to a storage address
	Addr string

	// addr stores the parsed storage address
	addr storage.Address

	// Path is the path to the content within a swarm manifest
	Path string
}

// Parse parses rawuri into a URI struct, where rawuri is expected to have one
// of the following formats:
//
// * <scheme>:/
// * <scheme>:/<addr>
// * <scheme>:/<addr>/<path>
// * <scheme>://
// * <scheme>://<addr>
// * <scheme>://<addr>/<path>
//
// with scheme one of bzz, bzz-raw, bzz-immutable, bzz-list or bzz-hash
func Parse(rawuri string) (*URI, error) {
	u, err := url.Parse(rawuri)
	if err != nil {
		return nil, err
	}
	uri := &URI{Scheme: u.Scheme}

	// check the scheme is valid
	switch uri.Scheme {
	case "bzz", "bzz-raw", "bzz-immutable", "bzz-list", "bzz-hash", "bzz-resource":
	default:
		return nil, fmt.Errorf("unknown scheme %q", u.Scheme)
	}

	// handle URIs like bzz://<addr>/<path> where the addr and path
	// have already been split by url.Parse
	if u.Host != "" {
		uri.Addr = u.Host
		uri.Path = strings.TrimLeft(u.Path, "/")
		return uri, nil
	}

	// URI is like bzz:/<addr>/<path> so split the addr and path from
	// the raw path (which will be /<addr>/<path>)
	parts := strings.SplitN(strings.TrimLeft(u.Path, "/"), "/", 2)
	uri.Addr = parts[0]
	if len(parts) == 2 {
		uri.Path = parts[1]
	}
	return uri, nil
}
func (u *URI) Resource() bool {
	return u.Scheme == "bzz-resource"
}

func (u *URI) Raw() bool {
	return u.Scheme == "bzz-raw"
}

func (u *URI) Immutable() bool {
	return u.Scheme == "bzz-immutable"
}

func (u *URI) List() bool {
	return u.Scheme == "bzz-list"
}

func (u *URI) Hash() bool {
	return u.Scheme == "bzz-hash"
}

func (u *URI) String() string {
	return u.Scheme + ":/" + u.Addr + "/" + u.Path
}

func (u *URI) Address() storage.Address {
	if u.addr != nil {
		return u.addr
	}
	if hashMatcher.MatchString(u.Addr) {
		u.addr = common.Hex2Bytes(u.Addr)
		return u.addr
	}
	return nil
}
