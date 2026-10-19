package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxMessageLength es el máximo de caracteres de un mensaje (el mismo límite que tiene el input del frontend)
const MaxMessageLength = 200

// errores propios del dominio: el resto del programa los compara para saber qué pasó
var (
	ErrEmptyMessage   = errors.New("el mensaje está vacío")
	ErrMessageTooLong = errors.New("el mensaje es demasiado largo")
)

// Message es un mensaje de chat dentro de la sala de un evento
//parecido a una clase tradicionañ
type Message struct {
	EventID  string
	UserID   string
	Username string
	Text     string
	SentAt   time.Time
}

// NewMessage es la única forma de crear un mensaje válido: aquí viven las reglas
func NewMessage(eventID, userID, username, text string) (Message, error) {
	//quita lo espacios
	text = strings.TrimSpace(text)

	if text == "" {
		return Message{}, ErrEmptyMessage
	}
    //para contra los emjos
	if utf8.RuneCountInString(text) > MaxMessageLength {
		return Message{}, ErrMessageTooLong
	}

	return Message{
		EventID:  eventID,
		UserID:   userID,
		Username: username,
		Text:     text,
		SentAt:   time.Now().UTC(),
	}, nil
}