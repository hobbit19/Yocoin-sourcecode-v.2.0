// Authored and revised by YOC team, 2018
// License placeholder #1

// Package deps contains the console JavaScript dependencies Go embedded.
package deps

//go:generate go-bindata -nometadata -pkg deps -o bindata.go bignumber.js
//go:generate gofmt -w -s bindata.go
