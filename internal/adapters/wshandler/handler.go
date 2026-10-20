package wshandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"horizon/flow/internal/domain"
	"horizon/flow/internal/ports"
	"horizon/flow/internal/service"
)

// cada cuánto le mandamos un ping al navegador para que el gateway
// no cierre la conexión por inactividad (YARP corta a los ~100 s)
const pingInterval = 30 * time.Second

// Handler atiende las conexiones WebSocket del chat
type Handler struct {
	chat           *service.ChatService
	allowedOrigins []string
}

func New(chat *service.ChatService, allowedOrigins []string) *Handler {
	return &Handler{chat: chat, allowedOrigins: allowedOrigins}
}

// incoming es el JSON que manda el navegador: {"text": "hola"}
type incoming struct {
	Text string `json:"text"`
}

// outgoing es un mensaje del chat que le mandamos al navegador
type outgoing struct {
	Type     string    `json:"type"`
	UserID   string    `json:"userId"`
	Username string    `json:"username"`
	Text     string    `json:"text"`
	SentAt   time.Time `json:"sentAt"`
}

// errorFrame es el aviso de error para quien lo provocó
type errorFrame struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

func toOutgoing(m domain.Message) outgoing {
	return outgoing{
		Type:     "message",
		UserID:   m.UserID,
		Username: m.Username,
		Text:     m.Text,
		SentAt:   m.SentAt,
	}
}

// Connect atiende GET /ws/{id}
func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	// el token viaja en la cookie; si no hay, entra como anónimo
	token := ""
	cookie, err := r.Cookie("horizon_token")
	if err == nil {
		token = cookie.Value
	}

	client, err := h.chat.Join(r.Context(), eventID, token)
	if err != nil {
		log.Printf("no se pudo entrar: evento=%s error=%v", eventID, err)
		writeJoinError(w, err)
		return
	}
	defer h.chat.Leave(eventID, client)

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.allowedOrigins,
	})
	if err != nil {
		return // Accept ya le respondió el error al navegador
	}
	defer conn.CloseNow()

	// ningún mensaje del navegador puede pesar más de 2 KB
	conn.SetReadLimit(2048)

	// cable de esta conexión: quien termine primero (lector o escritor) lo apaga
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// goroutine que lee lo que escribe el navegador
	go func() {
		defer cancel()
		h.readLoop(ctx, conn, eventID, client)
	}()

	// el ticker deja caer un valor en ticker.C cada pingInterval
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	// este for escribe hacia el navegador
	for {
		select {
		case <-ctx.Done():
			return

		case <-client.Done():
			// la sala nos sacó por ir muy lentos: cerramos la conexión
			conn.Close(websocket.StatusPolicyViolation, "conexión demasiado lenta")
			return

		case <-ticker.C:
			// Ping espera el pong, pero con un máximo de 10 s
			pingCtx, cancelPing := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pingCtx)
			cancelPing()
			if err != nil {
				return // el navegador no contestó: la conexión está muerta
			}

		case msg := <-client.Outbox:
			err := wsjson.Write(ctx, conn, toOutgoing(msg))
			if err != nil {
				return
			}
		}
	}
}

// readLoop lee los mensajes del navegador y se los pasa al service
func (h *Handler) readLoop(ctx context.Context, conn *websocket.Conn, eventID string, client *domain.Client) {
	for {
		var in incoming
		err := wsjson.Read(ctx, conn, &in)
		if err != nil {
			return // se cerró la conexión, o mandaron algo que no es JSON
		}

		err = h.chat.Send(ctx, eventID, client, in.Text)
		if err != nil {
			frame := errorFrame{Type: "error", Error: userMessage(err)}
			err = wsjson.Write(ctx, conn, frame)
			if err != nil {
				return
			}
		}
	}
}

// userMessage decide qué error se le muestra al usuario
func userMessage(err error) string {
	if errors.Is(err, service.ErrNotAuthenticated) ||
		errors.Is(err, domain.ErrEmptyMessage) ||
		errors.Is(err, domain.ErrMessageTooLong) {
		return err.Error()
	}
	return "no se pudo enviar el mensaje"
}

// writeJoinError traduce los errores del service a respuestas HTTP
func writeJoinError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrEventNotFound):
		http.Error(w, "el evento no existe", http.StatusNotFound)
	case errors.Is(err, ports.ErrInvalidToken):
		http.Error(w, "token inválido o expirado", http.StatusUnauthorized)
	case errors.Is(err, service.ErrChatClosed):
		http.Error(w, "el chat de este evento está cerrado", http.StatusGone)
	default:
		http.Error(w, "no se pudo entrar al chat", http.StatusBadGateway)
	}
}