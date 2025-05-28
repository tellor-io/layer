package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
}

type DiscordMessage struct {
	Content string `json:"content"`
}

func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{},
	}
}

func (n *DiscordNotifier) SendAlert(message string) error {
	payload := DiscordMessage{
		Content: message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling discord message: %w", err)
	}

	req, err := http.NewRequest("POST", n.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending discord message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord webhook returned non-200 status code: %d", resp.StatusCode)
	}

	return nil
}
