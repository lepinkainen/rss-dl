package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	RSSURL    string        `yaml:"rss_url"`
	OutputDir string        `yaml:"output_dir"`
	Discord   DiscordConfig `yaml:"discord"`
}

type DiscordConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Enabled    bool   `yaml:"enabled"`
	Username   string `yaml:"username"`
	AvatarURL  string `yaml:"avatar_url"`
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
		if err := downloadFile(cfg, item.Link); err != nil {
			log.WithFields(log.Fields{
				"url":   item.Link,
				"error": err,
			}).Error("Failed to download file")
			continue
		}
	}
}

// Download the file
func downloadFile(cfg Config, url string) error {
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

	path := filepath.Join(cfg.OutputDir, fileName)
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

	// Send Discord notification
	sendDiscordNotification(cfg, fileName)

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

// Discord webhook payload structures
type DiscordWebhook struct {
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds"`
}

type DiscordEmbed struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Color       int       `json:"color"`
	Timestamp   time.Time `json:"timestamp"`
}

// Send Discord notification for successful torrent download
func sendDiscordNotification(cfg Config, filename string) {
	if !cfg.Discord.Enabled || cfg.Discord.WebhookURL == "" {
		return
	}

	// Clean filename (remove .torrent extension if present)
	displayName := strings.TrimSuffix(filename, ".torrent")

	embed := DiscordEmbed{
		Title:       "🧲 Torrent Downloaded Successfully",
		Description: fmt.Sprintf("**%s**", displayName),
		Color:       0x00ff00, // Green color
		Timestamp:   time.Now(),
	}

	payload := DiscordWebhook{
		Username:  cfg.Discord.Username,
		AvatarURL: cfg.Discord.AvatarURL,
		Embeds:    []DiscordEmbed{embed},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to marshal Discord webhook payload")
		return
	}

	resp, err := http.Post(cfg.Discord.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to send Discord webhook")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
			"filename":    filename,
		}).Error("Discord webhook returned non-success status")
		return
	}

	log.WithFields(log.Fields{
		"filename": filename,
	}).Info("Discord notification sent successfully")
}
