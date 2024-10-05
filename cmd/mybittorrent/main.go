package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

var _ = json.Marshal

func decodeString(bString string, start int) (string, int, error) {
	var firstColonIndex int

	for i := start; i < len(bString); i++ {
		if bString[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := bString[start:firstColonIndex]

	lengthOfString, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", start, err
	}

	return bString[firstColonIndex+1 : firstColonIndex+1+lengthOfString], firstColonIndex + lengthOfString + 1, nil
}

func decodeInt(bString string, start int) (int, int, error) {
	if start == len(bString) {
		return start, 0, fmt.Errorf("Bad bencoded Int")
	}

	lastIdxOfInt := start

	for lastIdxOfInt < len(bString) && bString[lastIdxOfInt] != 'e' {
		lastIdxOfInt++
	}
	if lastIdxOfInt >= len(bString) || bString[lastIdxOfInt] != 'e' {
		return start, 0, fmt.Errorf("Bad bencoded Int")
	}

	lastIdxOfInt++
	dInt, err := strconv.Atoi(bString[start+1 : lastIdxOfInt-1])

	return dInt, lastIdxOfInt, err
}

func decodeList(bString string, start int) (interface{}, int, error) {
	// l4:hello3atuli3ee -> ["hello", "atul", 3]
	decodedList := make([]interface{}, 0)
	if start >= len(bString) {
		return decodedList, start, fmt.Errorf("Invalid bencoded list")
	}
	currIdx := start
	currIdx++

	for {
		if currIdx >= len(bString) {
			return nil, currIdx, fmt.Errorf("bad bencoded list")
		}

		if bString[currIdx] == 'e' {
			break
		}

		decodedVal, nextStartIdx, err := decodeBencode(bString, currIdx)
		if err != nil {
			return nil, start, err
		}
		currIdx = nextStartIdx

		decodedList = append(decodedList, decodedVal)

	}

	return decodedList, currIdx, nil
}

func decodeBencode(bencodedString string, start int) (interface{}, int, error) {
	switch {
	case unicode.IsDigit(rune(bencodedString[start])):
		return decodeString(bencodedString, start)
	case bencodedString[start] == 'i':
		return decodeInt(bencodedString, start)
	case bencodedString[0] == 'l':
		return decodeList(bencodedString, start)
	default:
		return "", len(bencodedString), fmt.Errorf("Only strings are supported at the moment")
	}
}

func main() {
	command := os.Args[1]

	if command == "decode" {

		bencodedValue := os.Args[2]

		decoded, _ , err := decodeBencode(bencodedValue, 0)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
