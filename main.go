package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
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
	// Configure logging
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:    true,
		TimestampFormat:  "2006-01-02 15:04:05",
		DisableTimestamp: false,
		DisableSorting:   true,
		// Make it more readable without the extra formatting
		DisableLevelTruncation: true,
		PadLevelText:           true,
	})

	// Get the path to the directory of the current executable
	ex, err := os.Executable()
	if err != nil {
		log.Fatal("Error getting executable path: ", err)
	}
	exPath := filepath.Dir(ex)

	// Read and parse config.yaml
	data, err := os.ReadFile(filepath.Join(exPath, "config.yaml"))
	if err != nil {
		log.Fatal("Error reading config file: ", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatal("Error parsing config file: ", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		log.Fatal("Error creating output directory: ", err)
	}

	resp, err := http.Get(cfg.RSSURL)
	if err != nil {
		log.Fatal("Error fetching RSS: ", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading RSS body: ", err)
	}

	var rss RSS
	err = xml.Unmarshal(body, &rss)
	if err != nil {
		log.Fatal("Error parsing RSS: ", err)
	}

	for _, item := range rss.Channel.Items {
		if err := downloadFile(cfg.OutputDir, item.Link); err != nil {
			log.WithFields(log.Fields{
				"url":   item.Link,
				"error": err,
			}).Error("Failed to download file")
			continue
		}
	}
}

// Download the file
func downloadFile(outputDir string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Check if content type is acceptable (optional)
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/") {
		return fmt.Errorf("unexpected content type: %s", contentType)
	}

	fileName := extractFileName(resp)
	if fileName == "" {
		return fmt.Errorf("could not determine filename")
	}

	path := filepath.Join(outputDir, fileName)
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	log.WithFields(log.Fields{
		"path": path,
	}).Info("Successfully downloaded file")
	return nil
}

// Helper function to extract filename
func extractFileName(resp *http.Response) string {
	// Try Content-Disposition first
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if parts := strings.Split(cd, "filename="); len(parts) > 1 {
			return strings.Trim(parts[1], "\"")
		}
	}

	// Fallback to URL path
	if url := resp.Request.URL; url != nil {
		return filepath.Base(url.Path)
	}

	return ""
}
