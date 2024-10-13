package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/jackpal/bencode-go"
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

func getPeers(filePath string) []string {
	decBencode, err := readTorrentFile(filePath)
	if err != nil {
		fmt.Println("Error decoding torrent file")
	}
	trackerURL := decBencode["announce"]
	torrInfo := decBencode["info"].(map[string]interface{})
	infoHashBytes, _ := extractInfoHash(torrInfo)

	trackerResp, err := getTrackerInfo(trackerURL.(string), infoHashBytes, "00112233445566778899", 6881, 0, 0, torrInfo["length"].(int), 1)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	dresp, _, err := decodeBencode(trackerResp, 0)
	decodedTrackerResp := dresp.(map[string]interface{})
	peerBytes := []byte(decodedTrackerResp["peers"].(string))
	if len(peerBytes)%6 != 0 {
		fmt.Println("Bad Peer received")
		return nil
	}

	peers := make([]string, 0)

	for i := 0; i < len(peerBytes); i += 6 {
		peers = append(peers, fmt.Sprintf("%d.%d.%d.%d:%d", peerBytes[i], peerBytes[i+1], peerBytes[i+2], peerBytes[i+3], binary.BigEndian.Uint16(peerBytes[i+4:i+6])))
	}

	return peers
}

func handshakePeer(ip string, port string, infoHash string) ([]byte, error) {
	address := fmt.Sprintf("%s:%s", ip, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Error connecting to address %s", address)
	}

	defer conn.Close()

	var handshakeMsg []byte
	prtlen := byte(19)
	protoMsg := []byte("BitTorrent protocol")
	protoReservedBytes := make([]byte, 8)
	protoPeerId := []byte{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9}

	handshakeMsg = append([]byte{prtlen}, protoMsg...)
	handshakeMsg = append(handshakeMsg, protoReservedBytes...)
	handshakeMsg = append(handshakeMsg, []byte(infoHash)...)
	handshakeMsg = append(handshakeMsg, protoPeerId...)
	_, err = conn.Write(handshakeMsg)
	if err != nil {
		fmt.Println("Error while writing to TCP connection")
		return nil, err
	}

	replyHandshake := make([]byte, 68)
	_, err = conn.Read(replyHandshake)
	if err != nil {
		fmt.Println("failed to read:", err)
		return nil, err
	}

	return replyHandshake, nil
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
	peers := getPeers(os.Args[2])

	for i := 0; i < len(peers); i++ {
		fmt.Println(peers[i])
	}
}

func command_handshake() {
	filePath := os.Args[2]
	client := os.Args[3]
	parts := strings.Split(client, ":")
	clientIp, clientPort := parts[0], parts[1]

	decBencode, _ := readTorrentFile(filePath)
	torrInfo := decBencode["info"].(map[string]interface{})
	infoHashBytes, _ := extractInfoHash(torrInfo)

	resp, err := handshakePeer(clientIp, clientPort, string(infoHashBytes))
	if err != nil {
		fmt.Printf("Error while estrablishing handshake with peer %s:%s", clientIp, clientPort)
		return
	}

	fmt.Println("Peer ID:", hex.EncodeToString(resp[48:]))
	return
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
