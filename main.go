package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Items []Item `xml:"item"`
}

type Item struct {
	Link string `xml:"link"`
}

type Config struct {
	RSSURL    string `yaml:"rss_url"`
	OutputDir string `yaml:"output_dir"`
}

func main() {
	// Get the path to the directory of the current executable
	ex, err := os.Executable()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	exPath := filepath.Dir(ex)

	// Read and parse config.yaml
	data, err := ioutil.ReadFile(filepath.Join(exPath, "config.yaml"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	resp, err := http.Get(cfg.RSSURL)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	var rss RSS
	err = xml.Unmarshal(body, &rss)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, item := range rss.Channel.Items {
		downloadFile(cfg.OutputDir, item.Link)
	}
}

// Download the file
func downloadFile(outputDir string, url string) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error downloading file:", err)
		return
	}
	defer resp.Body.Close()

	// Get filename from headers
	cd := resp.Header.Get("Content-Disposition")
	if cd == "" {
		fmt.Println("Could not get Content-Disposition")
		return
	}

	parts := strings.Split(cd, "filename=")
	if len(parts) < 2 {
		fmt.Println("Could not parse filename")
		return
	}
	fileName := strings.Trim(parts[1], "\"")

	// Join outputDir and fileName to get full path
	path := filepath.Join(outputDir, fileName)

	out, err := os.Create(path)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Downloaded:", path)
}
