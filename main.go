package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	vosk "github.com/alphacep/vosk-api/go"
	"github.com/imkira/go-libav/avcodec"
	"github.com/imkira/go-libav/avformat"
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

		// Decodificar WebM para PCM com go-libav
		convertStart := time.Now()
		pcmData, err := decodeWebM(tmpFilePath)
		if err != nil {
			log.Printf("Erro na decodificação com go-libav: %v", err)
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

// decodeWebM decodifica WebM para PCM usando go-libav
func decodeWebM(filePath string) ([]byte, error) {
	// Abrir arquivo WebM
	ctx, err := avformat.OpenInput(filePath, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir arquivo: %v", err)
	}
	defer ctx.CloseInput()

	// Encontrar informações do stream
	if err := ctx.FindStreamInfo(nil); err != nil {
		return nil, fmt.Errorf("erro ao encontrar stream: %v", err)
	}

	// Encontrar stream de áudio
	var audioStream *avformat.Stream
	for _, stream := range ctx.Streams() {
		if stream.CodecParameters().MediaType() == avcodec.MediaTypeAudio {
			audioStream = stream
			break
		}
	}
	if audioStream == nil {
		return nil, fmt.Errorf("nenhum stream de áudio encontrado")
	}

	// Configurar codec
	codec := avcodec.FindDecoder(audioStream.CodecParameters().CodecID())
	if codec == nil {
		return nil, fmt.Errorf("codec não encontrado")
	}
	codecCtx, err := avcodec.NewContext(codec)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar contexto: %v", err)
	}
	defer codecCtx.Free()

	if err := codecCtx.SetParameters(audioStream.CodecParameters()); err != nil {
		return nil, fmt.Errorf("erro ao configurar parâmetros: %v", err)
	}
	if err := codecCtx.Open(codec, nil); err != nil {
		return nil, fmt.Errorf("erro ao abrir codec: %v", err)
	}

	// Buffer para PCM
	var pcmData []byte
	packet := avcodec.NewPacket()
	defer packet.Free()
	frame := avcodec.NewFrame()
	defer frame.Free()

	// Decodificar
	for ctx.ReadFrame(packet) == nil {
		if packet.StreamIndex() != audioStream.Index() {
			continue
		}
		if err := codecCtx.SendPacket(packet); err != nil {
			continue
		}
		for codecCtx.ReceiveFrame(frame) == nil {
			// Converter para PCM (16-bit, 16 kHz, mono)
			samples := frame.Data()[0]
			pcmData = append(pcmData, samples...)
		}
	}

	return pcmData, nil
}