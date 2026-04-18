package main

import "sync"

var jobs = make(map[string]*Job)
var mu sync.Mutex

type TRequest struct {
	BookID    string `json:"book_id"`
	TLanguage string `json:"target_language"`
}

type ARequest struct {
	BookID    string `json:"book_id"`
	TLanguage string `json:"target_language"`
	VoiceID   string `json:"voice_id"`
}

type Job struct {
	Status   string `json:"status"`
	Error    string `json:"error"`
	FilePath string `json:"file_path"`
	Progress string `json:"progress"`
}
