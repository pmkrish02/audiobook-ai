# AudioBook AI

Convert any English text into an audiobook in 22 Indian languages — upload text, translate via Sarvam AI, and generate audio with ElevenLabs voice synthesis. Supports async generation with job tracking.

## How It Works

1. **Upload** a text file
2. **Translate** into any of 22 Indian languages using Sarvam AI
3. **Generate** audio from the translation using ElevenLabs — runs in the background
4. **Poll** for status and **download** the MP3 when ready

## Tech Stack

- **Go** — HTTP server, goroutines for async processing, mutex for job state
- **Sarvam AI** — translation across 22 Indian languages
- **ElevenLabs API** — text-to-speech audio generation (eleven_v3 model)

## How to Run

1. Set your API keys:
```bash
export SARVAM_API_KEY="your-sarvam-api-key"
export ELEVEN_LABS="your-elevenlabs-api-key"
```

2. Create the data directory:
```bash
mkdir -p data/books
```

3. Run the server:
```bash
go mod tidy
go run main.go
```

Server starts on `:8080`.

## API Endpoints

### `GET /health`
```bash
curl http://localhost:8080/health
```

### `POST /upload` — Upload a text file
```bash
curl -X POST http://localhost:8080/upload \
  -F "file=@mybook.txt"
```
Returns `book_id` and `char_count`.

### `GET /book/{id}/chunks` — View text chunks
```bash
curl http://localhost:8080/book/YOUR_BOOK_ID/chunks
```

### `POST /translate` — Translate to target language
```bash
curl -X POST http://localhost:8080/translate \
  -H "Content-Type: application/json" \
  -d '{"book_id": "YOUR_BOOK_ID", "target_language": "te"}'
```
Translates all chunks via Sarvam AI. Saves to `data/translations/`.

Supported language codes: `hi` (Hindi), `te` (Telugu), `ta` (Tamil), `bn` (Bengali), `mr` (Marathi), `gu` (Gujarati), `kn` (Kannada), `ml` (Malayalam), `pa` (Punjabi), `or` (Odia), and 12 more.

### `POST /audiotranslate` — Generate audio (synchronous)
```bash
curl -X POST http://localhost:8080/audiotranslate \
  -H "Content-Type: application/json" \
  -d '{"book_id": "YOUR_BOOK_ID", "target_language": "te", "voice_id": "YOUR_VOICE_ID"}'
```
Blocks until audio is complete. For long texts, use `/generate` instead.

### `POST /generate` — Generate audio (async with job tracking)
```bash
curl -X POST http://localhost:8080/generate \
  -H "Content-Type: application/json" \
  -d '{"book_id": "YOUR_BOOK_ID", "target_language": "te", "voice_id": "YOUR_VOICE_ID"}'
```
Returns immediately with a `job_id`. Audio generation runs in a background goroutine.

### `GET /status/{id}` — Check job progress
```bash
curl http://localhost:8080/status/YOUR_JOB_ID
```
Response:
```json
{
  "status": "generating audio",
  "progress": "chunk 3 of 7",
  "error": "",
  "file_path": ""
}
```

### `GET /download/{id}` — Download completed audio
```bash
curl -o audiobook.mp3 http://localhost:8080/download/YOUR_JOB_ID
```
Returns the MP3 file. Only works when job status is `done`.

## Architecture

```
Upload:
    Text file → UUID assigned → saved to data/books/

Translation:
    Read book → chunk (900 chars, sentence-aware)
        → Translate each chunk via Sarvam AI
        → Join → save to data/translations/

Audio Generation (async):
    POST /generate → returns job_id immediately
        → Background goroutine:
            Read translation → chunk (2000 chars)
            → Convert each chunk to audio via ElevenLabs
            → Update job progress after each chunk
            → Concatenate audio bytes
            → Save MP3 to data/audios/
            → Mark job as done
        → Client polls GET /status/{id}
        → Client downloads GET /download/{id}
```

## Key Patterns

- **Async job queue** — `/generate` returns immediately, audio processing runs in a background goroutine with `context.WithoutCancel` so it survives client disconnect
- **Mutex-protected job state** — concurrent access to the jobs map is safe via `sync.Mutex`
- **Progress tracking** — job status updates per chunk so clients can monitor long-running generation
- **Sentence-aware chunking** — text is split on sentence boundaries, not mid-word, for better translation quality

## Future Improvements

- Concurrent chunk processing with goroutines and semaphore pattern
- PDF and EPUB upload support
- Web UI for uploading, selecting language, and downloading audio
- Voice cloning — upload a voice sample, generate audiobook in that voice
- Persistent job storage (currently in-memory, lost on restart)
- Cost tracking per audiobook generation