package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"horizon/flow/internal/adapters/eventsclient"
	"horizon/flow/internal/adapters/jwtvalidator"
	"horizon/flow/internal/adapters/redisbroker"
	"horizon/flow/internal/adapters/wshandler"
	"horizon/flow/internal/service"
)

func main() {
	// 1. configuración: en Docker llega por variables de entorno;
	// en tu máquina se usan los valores por defecto
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	eventsURL := getenv("EVENTS_URL", "http://localhost:5006")
	publicKeyPath := getenv("JWT_PUBLIC_KEY_PATH", "../Horizon.Events/Security/Keys/public.pem")
	allowedOrigins := strings.Split(getenv("ALLOWED_ORIGINS", "localhost:3000"), ",")
	port := getenv("PORT", "5007")

	// 2. adapters de salida
	broker := redisbroker.New(redisAddr)
	events := eventsclient.New(eventsURL)
	tokens, err := jwtvalidator.New(publicKeyPath)
	if err != nil {
		log.Fatalf("no se pudo cargar la clave pública: %v", err)
	}

	// 3. el service recibe los adapters a través de sus ports
	chat := service.NewChatService(broker, tokens, events)

	// 4. adapter de entrada
	ws := wshandler.New(chat, allowedOrigins)

	// 5. rutas
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /ws/{id}", ws.Connect)

	log.Println("Flow escuchando en :" + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}