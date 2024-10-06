package main

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
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
	decBencode, _ := readTorrentFile(os.Args[2])
	torrInfo := decBencode["info"].(map[string]interface{})
	infoHashBytes, _ := extractInfoHash(torrInfo)
	infoHashString := hex.EncodeToString(infoHashBytes)

	fmt.Println("Tracker URL:", decBencode["announce"])
	fmt.Println("Length:", torrInfo["length"])
	fmt.Println("Info Hash:", infoHashString)
	fmt.Println("Piece Length:", torrInfo["piece length"])
	fmt.Println("Piece Hashes:", hex.EncodeToString([]byte(torrInfo["pieces"].(string))))
}

func command_peer() {
	decBencode, err := readTorrentFile(os.Args[2])
	if err != nil {
		fmt.Println("Error decoding torrent file")
	}
	trackerURL := decBencode["announce"]
	torrInfo := decBencode["info"].(map[string]interface{})
	infoHashBytes, _ := extractInfoHash(torrInfo)

	trackerResp, err := getTrackerInfo(trackerURL.(string), infoHashBytes, "00112233445566778899", 6881, 0, 0, torrInfo["length"].(int), 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	dresp, _, err := decodeBencode(trackerResp, 0)
	decodedTrackerResp := dresp.(map[string]interface{})
	peerBytes := []byte(decodedTrackerResp["peers"].(string))
	if len(peerBytes)%6 != 0 {
		fmt.Println("Bad Peer received")
		return
	}

	for i := 0; i < len(peerBytes); i += 6 {
		fmt.Print(fmt.Sprintf("%d.%d.%d.%d.%d", peerBytes[i], peerBytes[i+1], peerBytes[i+2], peerBytes[i+3], binary.BigEndian.Uint16(peerBytes[i+4:i+6])))
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
	default:
		fmt.Println("Unknow command:", command)
		os.Exit(1)
	}
}
