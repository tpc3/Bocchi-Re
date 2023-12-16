package utils

import "strings"

func TrimPrefix(msg string, prefix string, mention string) (bool, string, string) {
	isCmd := false
	var trimmedMsg string
	if strings.HasPrefix(msg, prefix) {
		isCmd = true
		trimmedMsg = strings.TrimPrefix(msg, prefix)
	} else if strings.HasPrefix(msg, mention) {
		isCmd = true
		trimmedMsg = strings.TrimPrefix(msg, mention)
		trimmedMsg = strings.TrimPrefix(trimmedMsg, " ")
	}

	splitMsg := strings.SplitN(trimmedMsg, " ", 2)
	var param string
	if len(splitMsg) == 2 {
		param = splitMsg[1]
	} else {
		param = ""
	}

	return isCmd, splitMsg[0], param
}
