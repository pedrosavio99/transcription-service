FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /transcription-service

FROM ubuntu:22.04
RUN apt-get update && apt-get install -y --no-install-recommends \
    libsndfile1 \
    libavcodec-dev \
    libavformat-dev \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /transcription-service .
COPY models/ ./models/
COPY templates/ ./templates/
EXPOSE 8080
CMD ["/app/transcription-service"]