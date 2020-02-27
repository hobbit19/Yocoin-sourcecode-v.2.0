package nov2019

import "strings"

func ezAddress(input string) string {
	str := strings.ToLower(input)
	if len(str) == 42 && strings.HasPrefix(str, "0x") {
		str = strings.Replace(str, "0x", "", 1)
	}
	return str
}
