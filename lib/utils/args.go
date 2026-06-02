package utils

import (
	"errors"
	"strings"
)

var ErrInvalidArg = errors.New("invalid arg")

func SplitCommandArgs(input string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false

	for _, r := range input {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if quote != 0 {
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			continue
		}

		switch r {
		case '\'', '"':
			quote = r
		case ' ', '\t', '\r', '\n':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, ErrInvalidArg
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args, nil
}

func CommandArgValue(args []string, index int) (string, bool) {
	if index+1 >= len(args) {
		return "", false
	}
	return args[index+1], true
}