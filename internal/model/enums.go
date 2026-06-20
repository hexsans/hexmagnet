package model

import "strings"

//revive:disable:line-length-limit

//go:generate go run github.com/abice/go-enum --marshal --names --nocase --nocomments --sql --sqlnullstr --values -t enums.gql.tmpl -f content_type.go -f facet_logic.go -f file_type.go

func namesToLower(names ...string) []string {
	result := make([]string, len(names))
	for i, name := range names {
		result[i] = strings.ToLower(name)
	}

	return result
}
