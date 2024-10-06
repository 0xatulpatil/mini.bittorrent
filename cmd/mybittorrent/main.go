package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"

	bencode "github.com/jackpal/bencode-go"
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
			return nil, currIdx, err
		}
		currIdx = nextStartIdx

		decodedList = append(decodedList, decodedVal)
	}

	return decodedList, currIdx + 1, nil
}

func decodeDict(bString string, start int) (map[string]interface{}, int, error) {
	// d3:foo3:bar5:helloi52ee
	dDict := make(map[string]interface{})
	currIdx := start + 1
	var key string
	key = ""

	if currIdx >= len(bString) {
		return nil, currIdx, fmt.Errorf("bad bencoded dict")
	}

	for {
		if currIdx >= len(bString) {
			return nil, currIdx, fmt.Errorf("bad bencoded dict")
		}
		if bString[currIdx] == 'e' {
			break
		}

		decodedVal, nextStartIdx, err := decodeBencode(bString, currIdx)
		if err != nil {
			return nil, currIdx, fmt.Errorf("bad bencoded dict")
		}

		if key == "" {
			ok := true
			key, ok = decodedVal.(string)
			if !ok {
				return nil, currIdx, fmt.Errorf("dict key is not a string")
			}
		} else {
			dDict[key] = decodedVal
			key = ""
		}

		currIdx = nextStartIdx
	}

	return dDict, currIdx + 1, nil
}

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

func command_decode() {
	bencodedValue := os.Args[2]

	decoded, _, err := decodeBencode(bencodedValue, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	jsonOutput, _ := json.Marshal(decoded)
	fmt.Println(string(jsonOutput))
}

func command_info() {
	filePath := os.Args[2]

	file, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error parsing file")
		os.Exit(1)
	}
	fileread := string(file)

	db, _, err := decodeBencode(fileread, 0)
	if err != nil {
		fmt.Println("Error Decoding Bencoded file")
		return
	}
	decBencode, _ := db.(map[string]interface{})
	torrInfo := decBencode["info"].(map[string]interface{})

	var infoHashReader bytes.Buffer
	err = bencode.Marshal(&infoHashReader, torrInfo)
	if err != nil {
		fmt.Print(err)
		return
	}
	infoHashString := infoHashReader.String()

	hasher := sha1.New()
	hasher.Write([]byte(infoHashString))
	sh1 := hasher.Sum(nil)
	infoHashString = hex.EncodeToString(sh1)

	fmt.Println("Tracker URL:", decBencode["announce"])
	fmt.Println("Length:", torrInfo["length"])
	fmt.Println("Info Hash:", infoHashString)
}

func main() {
	command := os.Args[1]

	switch command {
	case "decode":
		command_decode()
	case "info":
		command_info()
	default:
		fmt.Println("Unknow command:", command)
		os.Exit(1)
	}
}
