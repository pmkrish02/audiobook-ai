package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func chunkText(text string, chunkSize int) []string {
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

type TRequest struct {
	BookID    string `json:"book_id"`
	TLanguage string `json:"target_language"`
}

type ARequest struct{
	BookID    string `json:"book_id"`
	TLanguage string `json:"target_language"`
	VoiceID string `json:"voice_id"`

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
    	"text":     text,
    	"model_id": "eleven_v3",
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
func FindHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Could not find with this id ", http.StatusBadRequest)
		return
	}
	dirPath := "./data/books/"
	fileName := id + ".txt"
	filePath := filepath.Join(dirPath, fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "could not readfilepath", http.StatusBadRequest)
		return
	}
	chunks := chunkText(string(data), 2000)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chunks)

}
func AudioHandler(w http.ResponseWriter,r *http.Request){
	var audiorequest ARequest
	err := json.NewDecoder(r.Body).Decode(&audiorequest)
	if err != nil {
		http.Error(w, "error in reading json file", http.StatusInternalServerError)
		return
	}
	dirName := "./data/translations/"
	fileName := audiorequest.BookID + "_" + audiorequest.TLanguage + ".txt"
	filePath := filepath.Join(dirName, fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "error in going to the file", http.StatusInternalServerError)
		return
	}
	chunks := chunkText(string(data), 2000)
	var audiotranslated []byte
	for _, c := range chunks {
		caudio, err := audiotranslate(r.Context(), c, audiorequest.VoiceID)
		if err != nil {
			http.Error(w, fmt.Sprintf("error reading file: %v", err), http.StatusInternalServerError)
			return
		}
		audiotranslated = append(audiotranslated,caudio...)
	}
	audioDir := filepath.Join(".", "data", "audios")
	audioFile := audiorequest.BookID + "_" + audiorequest.TLanguage + ".mp3"
	
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		http.Error(w, "Could not read", http.StatusBadRequest)
		return
	}
	audiofilePath := filepath.Join(audioDir, audioFile)
	err = os.WriteFile(audiofilePath, audiotranslated, 0644)
	if err != nil {
		http.Error(w, "Could not read", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"translated_audio": "created"})

}
func TranslateHandler(w http.ResponseWriter, r *http.Request) {
	var request TRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "error in reading json file", http.StatusInternalServerError)
		return
	}
	dirName := "./data/books/"
	fileName := request.BookID + ".txt"
	filePath := filepath.Join(dirName, fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "error in reading the file", http.StatusInternalServerError)
		return
	}
	chunks := chunkText(string(data), 2000)
	var translated []string
	for _, c := range chunks {
		ttext, err := translate(r.Context(), c, request.TLanguage)
		if err != nil {
			http.Error(w, fmt.Sprintf("error reading file: %v", err), http.StatusInternalServerError)
			return
		}
		translated = append(translated, ttext)
	}

	fulltext := strings.Join(translated, " ")
	transDir := filepath.Join(".", "data", "translations")
	transFile := request.BookID + "_" + request.TLanguage + ".txt"
	if err := os.MkdirAll(transDir, 0755); err != nil {
		http.Error(w, "Could not read", http.StatusBadRequest)
		return
	}
	transfilePath := filepath.Join(transDir, transFile)
	err = os.WriteFile(transfilePath, []byte(fulltext), 0644)
	if err != nil {
		http.Error(w, "Could not write into the file", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"translated_text": fulltext})

}
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bad File", http.StatusBadRequest)
		return
	}
	defer file.Close()
	if header.Filename == "" {
		http.Error(w, "filename can't be empty", http.StatusBadRequest)
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Could not read Bad File", http.StatusBadRequest)
		return
	}

	newuuid := uuid.New().String()
	fileName := newuuid + ".txt"
	dirPath := filepath.Join(".", "data/books")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		http.Error(w, "i have error", http.StatusBadRequest)
		return

	}
	filePath := filepath.Join(dirPath, fileName)
	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		http.Error(w, "Could not read", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"book_id": newuuid, "char_count": len(content)})

}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
func main() {
	router := http.NewServeMux()
	router.HandleFunc("GET /health", HealthHandler)
	router.HandleFunc("POST /upload", UploadHandler)
	router.HandleFunc("POST /translate", TranslateHandler)
	router.HandleFunc("GET /book/{id}/chunks", FindHandler)
	router.HandleFunc("POST /audiotranslate",AudioHandler)
	fmt.Println("starting server on the port 8080")
	fmt.Println("SARVAM KEY loaded:", os.Getenv("SARVAM_API_KEY") != "")
	fmt.Println("ELEVEN_LABS key loaded:", os.Getenv("ELEVEN_LABS") != "")
	http.ListenAndServe(":8080", router)
}

