package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	router := http.NewServeMux()
	router.HandleFunc("GET /health", HealthHandler)
	router.HandleFunc("POST /upload", UploadHandler)
	router.HandleFunc("POST /translate", TranslateHandler)
	router.HandleFunc("GET /book/{id}/chunks", FindHandler)
	router.HandleFunc("POST /audiotranslate", AudioHandler)
	router.HandleFunc("POST /generate", GenerateHandler)
	router.HandleFunc("GET /status/{id}", StatusHandler)
	router.HandleFunc("GET /download/{id}", DownloadHandler)
	router.HandleFunc("POST /voice", VoiceCloning)
	fmt.Println("starting server on the port 8080")
	fmt.Println("SARVAM KEY loaded:", os.Getenv("SARVAM_API_KEY") != "")
	fmt.Println("ELEVEN_LABS key loaded:", os.Getenv("ELEVEN_LABS") != "")
	http.ListenAndServe(":8080", router)
}
