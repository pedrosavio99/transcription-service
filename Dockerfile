# Usar imagem base com Go e dependências necessárias
FROM golang:1.21 AS builder

# Instalar dependências para FFmpeg
RUN apt-get update && apt-get install -y \
    ffmpeg \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Definir diretório de trabalho
WORKDIR /app

# Copiar go.mod e go.sum primeiro para aproveitar o cache
COPY go.mod go.sum ./

# Atualizar go.sum e baixar dependências
RUN go mod tidy && go mod download

# Copiar o restante do código
COPY . .

# Compilar o binário com CGO habilitado
RUN CGO_ENABLED=1 GOOS=linux GOFLAGS="-mod=readonly" go build -o /transcription-service

# Imagem final menor
FROM ubuntu:22.04

# Instalar FFmpeg na imagem final
RUN apt-get update && apt-get install -y \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

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