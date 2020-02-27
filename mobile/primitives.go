// Authored and revised by YOC team, 2016-2018
// License placeholder #1

// Contains various wrappers for primitive types.

package yocoin

import (
	"errors"
	"fmt"
)

// Strings represents s slice of strs.
type Strings struct{ strs []string }

// Size returns the number of strs in the slice.
func (s *Strings) Size() int {
	return len(s.strs)
}

// Get returns the string at the given index from the slice.
func (s *Strings) Get(index int) (str string, _ error) {
	if index < 0 || index >= len(s.strs) {
		return "", errors.New("index out of bounds")
	}
	return s.strs[index], nil
}

// Set sets the string at the given index in the slice.
func (s *Strings) Set(index int, str string) error {
	if index < 0 || index >= len(s.strs) {
		return errors.New("index out of bounds")
	}
	s.strs[index] = str
	return nil
}

// String implements the Stringer interface.
func (s *Strings) String() string {
	return fmt.Sprintf("%v", s.strs)
}
