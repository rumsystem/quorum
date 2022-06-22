package utils

import (
	"strings"
	"unicode"
)

func LongestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}

	prefix := strs[0]
	for i := 1; i < len(strs); i++ {
		for !strings.HasPrefix(strs[i], prefix) {
			prefix = prefix[0 : len(prefix)-1]
			if len(prefix) == 0 {
				return ""
			}
		}
	}

	return prefix
}

func LowerFirstLetter(s string) string {
	for i, v := range s {
		return string(unicode.ToLower(v)) + s[i+1:]
	}

	return s
}
