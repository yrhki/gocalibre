package main

import (
	"fmt"
)


func prompt(prefer bool, text string) bool {
	var input string

	if prefer {
		fmt.Printf(":: %s? [Y/n] ", text)
	} else {
		fmt.Printf(":: %s? [y/N] ", text)
	}

	if _, err := fmt.Scanln(&input); err != nil { return false }

	switch input {
	case "y", "Y":
		return true
	case "n", "N":
		return false
	case "":
		return prefer
	default:
		return prompt(prefer, text)
	}
}
