package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/alphacep/vosk-api/go"
)

func main() {
	// Configurar logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Inicializar Vosk
	vosk.SetLogLevel(0)
	modelPath := "./models/vosk-model-small-en-us-0.15"
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		log.Fatalf("Modelo Vosk não encontrado em %s", modelPath)
	}
	model, err := vosk.NewModel(modelPath)
	if err != nil {
		log.Fatalf("Erro ao carregar modelo Vosk: %v", err)
	}
	defer model.Free()

	// Inicializar Gin
	r := gin.Default()

	// Servir HTML estático
	r.Static("/static", "./templates")
	r.GET("/", func(c *gin.Context) {
		c.File("./templates/index.html")
	})

	// Rota de health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Rota de transcrição
	r.POST("/transcribe", func(c *gin.Context) {
		startTime := time.Now()
		lang := c.PostForm("lang")
		if lang != "en" {
			log.Printf("Idioma não suportado: %s, usando en", lang)
			lang = "en"
		}
		log.Printf("Iniciando transcrição para idioma: %s", lang)

		// Obter arquivo de áudio
		file, err := c.FormFile("file")
		if err != nil {
			log.Printf("Erro ao obter arquivo: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Falha ao obter arquivo"})
			return
		}

		// Verificar tamanho (máximo 5 MB)
		if file.Size > 5*1024*1024 {
			log.Printf("Arquivo muito grande: %d bytes", file.Size)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Arquivo excede o limite de 5 MB"})
			return
		}

		// Salvar arquivo temporário
		tmpDir := os.TempDir()
		tmpFilePath := filepath.Join(tmpDir, file.Filename)
		if err := c.SaveUploadedFile(file, tmpFilePath); err != nil {
			log.Printf("Erro ao salvar arquivo: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao salvar arquivo"})
			return
		}
		log.Printf("Arquivo temporário salvo: %s", tmpFilePath)
		defer os.Remove(tmpFilePath)

		// Decodificar WebM para PCM usando FFmpeg
		convertStart := time.Now()
		pcmData, err := decodeWebM(tmpFilePath)
		if err != nil {
			log.Printf("Erro na decodificação com FFmpeg: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro na decodificação de áudio: %v", err)})
			return
		}
		log.Printf("Decodificação concluída em %v", time.Since(convertStart))

		// Inicializar reconhecedor Vosk
		rec, err := vosk.NewRecognizer(model, 16000)
		if err != nil {
			log.Printf("Erro ao inicializar reconhecedor: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao inicializar reconhecedor: %v", err)})
			return
		}
		defer rec.Free()
		rec.SetWords(true)

		// Transcrever PCM
		log.Println("Iniciando transcrição")
		transcribeStart := time.Now()
		chunkSize := 4096
		for i := 0; i < len(pcmData); i += chunkSize {
			end := i + chunkSize
			if end > len(pcmData) {
				end = len(pcmData)
			}
			if !rec.AcceptWaveform(pcmData[i:end]) {
				continue
			}
		}

		// Obter resultado
		result := rec.FinalResult()
		log.Printf("Transcrição concluída em %v: %s", time.Since(transcribeStart), result)
		log.Printf("Tempo total: %v", time.Since(startTime))

		// Retornar resultado
		c.JSON(http.StatusOK, gin.H{
			"transcribed_text": result,
			"time_total_ms":    time.Since(startTime).Milliseconds(),
		})
	})

	// Iniciar servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Iniciando servidor na porta %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}

// decodeWebM usa FFmpeg para decodificar WebM para PCM
func decodeWebM(filePath string) ([]byte, error) {
	return decodeWithFFmpeg(filePath)
}

// decodeWithFFmpeg usa FFmpeg para converter o arquivo de entrada para WAV
func decodeWithFFmpeg(filePath string) ([]byte, error) {
	wavPath := filePath + ".wav"
	cmd := exec.Command(
		"ffmpeg",
		"-i", filePath,
		"-f", "wav",
		"-acodec", "pcm_s16le",
		"-ac", "1",
		"-ar", "16000",
		"-t", "30",
		"-loglevel", "error",
		"-threads", "1",
		wavPath,
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("erro no FFmpeg: %v", err)
	}
	defer os.Remove(wavPath)

	data, err := os.ReadFile(wavPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler WAV: %v", err)
	}
	return data, nil
}