package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	TunnelX "github.com/xtls/libxray"
	"github.com/xtls/libxray/nodep"
)

func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func downloadFileIfNotExists(url string, writePath string) error {
	if _, err := os.Stat(writePath); err == nil {
		fmt.Printf("File already exists: %s, skipping download.\n", writePath)
		return nil
	}

	client := http.Client{}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = nodep.WriteBytes(body, writePath)
	return err
}

func saveTimestamp(datDir string) error {
	ts := time.Now().Unix()
	tsText := strconv.FormatInt(ts, 10)
	tsPath := path.Join(datDir, "timestamp.txt")
	return nodep.WriteText(tsText, tsPath)
}

func parseCallResponse(text string) (nodep.CallResponse[string], error) {
	var response nodep.CallResponse[string]
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(decoded, &response)
	return response, err
}

func makeLoadGeoDataRequest(datDir string, name string, geoType string) (string, error) {
	var request TunnelX.CountGeoDataRequestVPNS
	request.DatDir = datDir
	request.Name = name
	request.GeoType = geoType

	data, err := json.Marshal(&request)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func downloadDat(url string, datDir string, fileName string, geoType string) {
	datFile := fmt.Sprintf("%s.dat", fileName)
	geositePath := path.Join(datDir, datFile)
	err := downloadFileIfNotExists(url, geositePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	geoReq, err := makeLoadGeoDataRequest(datDir, fileName, geoType)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	res := TunnelX.CountGeoDataVPNS(geoReq)
	resp, err := parseCallResponse(res)
	if err != nil || !resp.Success {
		fmt.Println("Failed to load geosite:", res)
		os.Exit(1)
	}
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	datDir := path.Join(cwd, "dat")
	err = ensureDir(datDir)
	if err != nil {
		fmt.Println("Failed to ensure directory:", err)
		os.Exit(1)
	}

	// Download geosite.dat
	downloadDat("https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat", datDir, "geosite", "domain")
	// Download geoip.dat
	downloadDat("https://github.com/v2fly/geoip/releases/latest/download/geoip.dat", datDir, "geoip", "ip")

	// Save timestamp
	err = saveTimestamp(datDir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Geo data setup completed successfully.")
}
