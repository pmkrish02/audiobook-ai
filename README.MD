# AudioBook AI

Convert any English text into an audiobook in 22 Indian languages — upload text, translate via Sarvam AI, and generate audio with ElevenLabs voice synthesis.

## How It Works

1. **Upload** a text file
2. **Translate** the text into any of 22 Indian languages (Hindi, Telugu, Tamil, Bengali, etc.) using Sarvam AI
3. **Generate audio** from the translated text using ElevenLabs text-to-speech
4. Get back an MP3 audiobook in your chosen language

## Tech Stack

- **Go** — HTTP server, file handling, chunking
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
Returns `book_id` and `char_count`. Use the `book_id` for all subsequent calls.

### `GET /book/{id}/chunks` — View text chunks
```bash
curl http://localhost:8080/book/YOUR_BOOK_ID/chunks
```
Returns the text split into chunks (2000 chars each, split on sentence boundaries).

### `POST /translate` — Translate to target language
```bash
curl -X POST http://localhost:8080/translate \
  -H "Content-Type: application/json" \
  -d '{"book_id": "YOUR_BOOK_ID", "target_language": "te"}'
```
Translates all chunks via Sarvam AI and saves the full translated text to `data/translations/`.

Supported language codes: `hi` (Hindi), `te` (Telugu), `ta` (Tamil), `bn` (Bengali), `mr` (Marathi), `gu` (Gujarati), `kn` (Kannada), `ml` (Malayalam), `pa` (Punjabi), `or` (Odia), and more — 22 Indian languages total.

### `POST /audiotranslate` — Generate audiobook
```bash
curl -X POST http://localhost:8080/audiotranslate \
  -H "Content-Type: application/json" \
  -d '{"book_id": "YOUR_BOOK_ID", "target_language": "te", "voice_id": "YOUR_ELEVENLABS_VOICE_ID"}'
```
Converts the translated text to audio using ElevenLabs and saves the MP3 to `data/audios/`.

## Architecture

```
Upload Flow:
    Text file → UUID assigned → saved to data/books/

Translation Flow:
    Read book by ID
        → Chunk text (2000 chars, sentence-aware)
        → Translate each chunk via Sarvam AI
        → Join translated chunks
        → Save to data/translations/

Audio Flow:
    Read translated file
        → Chunk text
        → Convert each chunk to audio via ElevenLabs
        → Concatenate audio bytes
        → Save MP3 to data/audios/
```

## Future Improvements

- Concurrent chunk processing with goroutines and semaphore pattern for faster translation and audio generation
- PDF and EPUB upload support
- Web UI for uploading, selecting language, and downloading audio
- Async job queue — return a job ID immediately, process in background, poll for status
- Voice cloning — upload a voice sample, generate audiobook in that voice
- Cost tracking per audiobook generation