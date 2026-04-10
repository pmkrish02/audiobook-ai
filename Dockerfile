FROM golang:1.25-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN go build -o audiobook main.go

FROM alpine:latest
WORKDIR /app
RUN mkdir -p /app/data/books
COPY --from=0 /app/audiobook /app/audiobook
EXPOSE 8080
CMD ["/app/audiobook"]