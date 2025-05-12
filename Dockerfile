# Usar imagem base com Go e dependências necessárias
FROM golang:1.21 AS builder

# Instalar dependências para FFmpeg e Vosk
RUN apt-get update && apt-get install -y \
    ffmpeg \
    pkg-config \
    wget \
    unzip \
    build-essential \
    && wget https://github.com/alphacep/vosk-api/releases/download/v0.3.45/vosk-linux-x86_64-0.3.45.zip && \
    unzip vosk-linux-x86_64-0.3.45.zip && \
    mv vosk-linux-x86_64-0.3.45/libvosk.so /usr/lib/ && \
    mv vosk-linux-x86_64-0.3.45/vosk_api.h /usr/include/ && \
    rm -rf vosk-linux-x86_64-0.3.45.zip vosk-linux-x86_64-0.3.45 && \
    apt-get remove -y wget unzip && \
    apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*

# Definir diretório de trabalho
WORKDIR /app

# Copiar go.mod e go.sum primeiro para aproveitar o cache
COPY go.mod go.sum ./

# Limpar cache de módulos e baixar dependências
RUN go clean -modcache && go mod tidy && go mod download

# Copiar o restante do código
COPY . .

# Compilar o binário com CGO habilitado
RUN CGO_ENABLED=1 GOOS=linux go build -o /transcription-service

# Imagem final menor
FROM ubuntu:22.04

# Instalar FFmpeg na imagem final
RUN apt-get update && apt-get install -y \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# Copiar libvosk.so da etapa de build
COPY --from=builder /usr/lib/libvosk.so /usr/lib/

# Copiar o binário da etapa de build
COPY --from=builder /transcription-service /transcription-service

# Copiar modelos Vosk e templates
COPY --from=builder /app/models /app/models
COPY --from=builder /app/templates /app/templates

# Definir diretório de trabalho
WORKDIR /app

# Expor a porta
EXPOSE 8080

# Comando para iniciar o serviço
CMD ["/transcription-service"]