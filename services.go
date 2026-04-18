package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func chunkText(text string, chunkSize int) []string {
	if text == "" {
		return nil
	}
	var chunks []string
	sentences := strings.Split(text, ".")
	currentchunk := ""
	for _, s := range sentences {
		candidate := currentchunk + s + "."
		if len(candidate) > chunkSize {
			chunks = append(chunks, currentchunk)
			currentchunk = s + "."
		} else {
			currentchunk = candidate
		}
	}
	if currentchunk != "" {
		chunks = append(chunks, currentchunk)
	}
	return chunks

}

func translate(ctx context.Context, text string, targetLang string) (string, error) {
	url := "https://api.sarvam.ai/translate"
	body := map[string]string{
		"input":                text,
		"source_language_code": "en-IN",
		"target_language_code": targetLang,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	payload := bytes.NewReader(jsonBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, payload)
	if err != nil {
		return "", fmt.Errorf("could not marshal: %w", err)
	}
	req.Header.Add("api-subscription-key", os.Getenv("SARVAM_API_KEY"))
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not marshal: %w", err)
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("could not read the res body: %w", err)
	}
	fmt.Println("Sarvam response:", string(resBody))
	var result map[string]any
	err = json.Unmarshal(resBody, &result)
	if err != nil {
		return "", err
	}
	translated, ok := result["translated_text"].(string)
	if !ok {
		return "", fmt.Errorf("no translated_text in response")
	}
	return translated, nil

}

func audiotranslate(ctx context.Context, text string, voiceid string) ([]byte, error) {
	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s?output_format=mp3_44100_128", voiceid)
	body := map[string]string{
		"text":          text,
		"model_id":      "eleven_v3",
		"language_code": "te",
	}
	jsonAudio, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	payload := bytes.NewReader(jsonAudio)
	req, err := http.NewRequestWithContext(ctx, "POST", url, payload)
	if err != nil {
		return nil, fmt.Errorf("could not create connection: %w", err)
	}
	req.Header.Add("xi-api-key", os.Getenv("ELEVEN_LABS"))
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get response: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("elevenlabs returned status %d", res.StatusCode)
	}
	resAudio, err := io.ReadAll(res.Body)

	if err != nil {
		return nil, fmt.Errorf("could not read the body: %w", err)
	}
	return resAudio, nil
}

func withRetry(maxRetries int, fn func() error) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil // it worked, stop retrying
		}
		wait := time.Duration(1<<i) * time.Second // 1s, 2s, 4s
		fmt.Printf("retry %d, waiting %v\n", i+1, wait)
		time.Sleep(wait)
	}
	return err // all retries failed, return last error
}
