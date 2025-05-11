module transcription-service

go 1.21

require (
	github.com/alphacep/vosk-api/go v0.3.45
	github.com/gin-gonic/gin v1.10.0
	github.com/giorgisio/go-libav v0.0.0-20200807153303-a94c026c2173
)

require (
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
)

// Substituir github.com/giorgisio/go-libav por github.com/cvley/go-libav
replace github.com/giorgisio/go-libav => github.com/cvley/go-libav v0.0.0-20200807153303-a94c026c2173

// Excluir commit problemático para evitar conflitos transitivos
exclude github.com/imkira/go-libav v0.0.0-20180115004737-6ea2b4c24598