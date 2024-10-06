package main

import (
	"bytes"
	"crypto/sha1"
	// "encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	bencode "github.com/jackpal/bencode-go"
)

func readTorrentFile(filePath string) (map[string]interface{}, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error parsing file")
		os.Exit(1)
	}
	fileread := string(file)

	db, _, err := decodeBencode(fileread, 0)
	if err != nil {
		return nil, fmt.Errorf("Error Decoding Bencoded file")
	}

	decBencode, _ := db.(map[string]interface{})

	return decBencode, nil
}

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

func extractInfoHash(torrInfo map[string]interface{}) ([]byte, error) {
	var infoHashReader bytes.Buffer
	err := bencode.Marshal(&infoHashReader, torrInfo)
	if err != nil {
		fmt.Print(err)
		return []byte{}, nil
	}
	infoHashString := infoHashReader.String()

	hasher := sha1.New()
	hasher.Write([]byte(infoHashString))

	return hasher.Sum(nil), nil
}

func getTrackerInfo(trackerBaseURL string,
	infoHash []byte,
	peerId string,
	port int,
	uploaded int,
	downloaded int,
	left int,
	compact int,
) (string, error) {
	params := url.Values{}
	params.Add("info_hash", string(infoHash))
	params.Add("peer_id", peerId)
	params.Add("port", strconv.Itoa(port))
	params.Add("uploaded", strconv.Itoa(uploaded))
	params.Add("downloaded", strconv.Itoa(downloaded))
	params.Add("left", strconv.Itoa(left))
	params.Add("compact", strconv.Itoa(compact))

	trackerURL := fmt.Sprintf("%s?%s", trackerBaseURL, params.Encode())

	resp, err := http.Get(trackerURL)
	if err != nil {
		return "", err
	}

	respBody, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}

	return string(respBody), nil
}