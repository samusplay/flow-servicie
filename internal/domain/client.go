package domain

import "sync"

const outboxSize = 32

// Client es una conexión de una persona dentro de una sala
type Client struct {
	ID       string
	UserID   string // vacío si es anónimo
	Username string
	Outbox   chan Message

	done      chan struct{}
	closeOnce sync.Once
}

func NewClient(id, userID, username string) *Client {
	return &Client{
		ID:       id,
		UserID:   userID,
		Username: username,
		Outbox:   make(chan Message, outboxSize),
		done:     make(chan struct{}),
	}
}

// Send intenta dejar el mensaje en la fila del cliente sin esperar.
// Devuelve false si la fila está llena (el cliente va muy lento).
func (c *Client) Send(msg Message) bool {
	select {
	case c.Outbox <- msg:
		return true
	default:
		return false
	}
}

// Done se cierra cuando el cliente debe desconectarse
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// Close avisa que el cliente debe desconectarse. Es seguro llamarlo varias veces.
func (c *Client) Close() {
	c.closeOnce.Do(func() { close(c.done) })
}