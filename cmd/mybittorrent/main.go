package main

import (
	"fmt"
	"os"
	"unicode"
)

func decodeBencode(bencodedString string, start int) (interface{}, int, error) {
	switch {
	case unicode.IsDigit(rune(bencodedString[start])):
		return decodeString(bencodedString, start)
	case bencodedString[start] == 'i':
		return decodeInt(bencodedString, start)
	case bencodedString[start] == 'l':
		return decodeList(bencodedString, start)
	case bencodedString[start] == 'd':
		return decodeDict(bencodedString, start)

	default:
		return "", len(bencodedString), fmt.Errorf("Only strings are supported at the moment")
	}
}

func main() {
	command := os.Args[1]

	switch command {
	case "decode":
		command_decode()
	case "info":
		command_info()
	case "peers":
		command_peer()
	case "handshake":
		command_handshake()
	default:
		fmt.Println("Unknow command:", command)
		os.Exit(1)
	}
}
