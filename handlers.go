package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

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
func AudioHandler(w http.ResponseWriter, r *http.Request) {
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
		audiotranslated = append(audiotranslated, caudio...)
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
	chunks := chunkText(string(data), 900)
	translated := make([]string, len(chunks))
	errors := make([]string, len(chunks))
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	for i, c := range chunks {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, c string) {
			defer wg.Done()
			defer func() { <-sem }()
			var ttext string
			err = withRetry(3, func() error {
				var err error
				ttext, err = translate(r.Context(), c, request.TLanguage)
				return err
			})
			if err != nil {
				errors[i] = fmt.Sprintf("Error in chunk %d: %v", i, err)
				return
			}
			translated[i] = ttext

		}(i, c)

	}
	wg.Wait()

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

func GenerateHandler(w http.ResponseWriter, r *http.Request) {
	var usereq ARequest
	err := json.NewDecoder(r.Body).Decode(&usereq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad JSON Request %v", err), http.StatusInternalServerError)
		return
	}
	jobID := uuid.New().String()
	mu.Lock()
	jobs[jobID] = &Job{Status: "starting"}
	mu.Unlock()
	bgCtx := context.WithoutCancel(r.Context())
	go func() {
		dirName := "./data/translations/"
		fileName := usereq.BookID + "_" + usereq.TLanguage + ".txt"
		filePath := filepath.Join(dirName, fileName)
		data, err := os.ReadFile(filePath)
		if err != nil {
			mu.Lock()
			jobs[jobID].Status = "failed"
			jobs[jobID].Error = err.Error()
			mu.Unlock()
			return
		}
		chunks := chunkText(string(data), 2000)
		audioBytes := make([][]byte, len(chunks))
		sem := make(chan struct{}, 2)
		var wg sync.WaitGroup
		for i, c := range chunks {
			mu.Lock()
			jobs[jobID].Status = "generating audio"
			jobs[jobID].Progress = fmt.Sprintf("chunk %d of %d", i+1, len(chunks))
			mu.Unlock()
			wg.Add(1)
			sem <- struct{}{}
			go func(i int, c string) {
				defer wg.Done()
				defer func() { <-sem }()
				var caudio []byte
				err := withRetry(3, func() error {
					var err error
					caudio, err = audiotranslate(bgCtx, c, usereq.VoiceID)
					return err
				})
				if err != nil {
					mu.Lock()
					jobs[jobID].Error = err.Error()
					mu.Unlock()
					return
				}
				audioBytes[i] = caudio
			}(i, c)

		}
		wg.Wait()
		var finalAudio []byte
		for _, chunk := range audioBytes {
			finalAudio = append(finalAudio, chunk...)
		}
		audioDir := filepath.Join(".", "data", "audios")
		os.MkdirAll(audioDir, 0755)
		audioFile := usereq.BookID + "_" + usereq.TLanguage + ".mp3"
		audioPath := filepath.Join(audioDir, audioFile)
		err = os.WriteFile(audioPath, finalAudio, 0644)
		if err != nil {
			mu.Lock()
			jobs[jobID].Status = "failed"
			jobs[jobID].Error = err.Error()
			mu.Unlock()
			return
		}

		mu.Lock()
		jobs[jobID].Status = "done"
		jobs[jobID].FilePath = audioPath
		mu.Unlock()
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"job_id": jobID, "status": "starting"})

}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	mu.Lock()
	userjob, ok := jobs[id]
	mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if userjob, ok = jobs[id]; ok {
		json.NewEncoder(w).Encode(userjob)
	} else {
		http.Error(w, "Could not find", http.StatusNotFound)
		return
	}
}
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	mu.Lock()
	userjob, ok := jobs[id]
	mu.Unlock()
	w.Header().Set("Content-Type", "application/octet-stream")
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if userjob.Status != "done" {
		http.Error(w, "audio still processing", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, userjob.FilePath)

}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func VoiceCloning(w http.ResponseWriter, r *http.Request) {
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
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "voice name is required", http.StatusBadRequest)
		return
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("files", header.Filename)
	if err != nil {
		http.Error(w, "Could not create the form file", http.StatusBadRequest)
		return
	}
	_, err = io.Copy(part, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to copy audio: %v", err), http.StatusInternalServerError)
		return
	}
	err = writer.WriteField("name", name)
	if err != nil {
		http.Error(w, "Could not write the form filed", http.StatusBadRequest)
		return
	}
	writer.Close()

	req, err := http.NewRequestWithContext(r.Context(), "POST", "https://api.elevenlabs.io/v1/voices/add", body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create request: %v", err), http.StatusInternalServerError)
		return
	}
	req.Header.Add("xi-api-key", os.Getenv("ELEVEN_LABS"))
	req.Header.Add("Content-Type", writer.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Could not make the client", http.StatusBadRequest)
		return
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "could not read response", http.StatusInternalServerError)
		return
	}
	fmt.Println("ElevenLabs response:", string(resBody))
	if res.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("elevenlabs returned status %d", res.StatusCode), http.StatusInternalServerError)
		return
	}
	var result map[string]any
	json.Unmarshal(resBody, &result)
	voiceID, ok := result["voice_id"].(string)
	if !ok {
		http.Error(w, "no voice_id in response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"voice_id": voiceID, "name": name})
}
