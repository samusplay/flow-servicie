package service

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"

	"horizon/flow/internal/domain"
	"horizon/flow/internal/ports"
)

var (
	// ErrChatClosed: el evento ya terminó, no se puede entrar al chat
	ErrChatClosed = errors.New("el chat de este evento está cerrado")

	// ErrNotAuthenticated: un anónimo intentó escribir
	ErrNotAuthenticated = errors.New("debes iniciar sesión para escribir")
)

// roomEntry une una sala con la suscripción al broker que la alimenta
type roomEntry struct {
	room   *domain.Room
	cancel context.CancelFunc // corta la suscripción cuando la sala queda vacía
}

// ChatService es la lógica del chat: usa los ports, no sabe qué hay detrás
//guardar en memoria las slaas activas en el sevridor
type ChatService struct {
	broker ports.MessageBroker
	tokens ports.TokenValidator
	events ports.EventLookup

	//candado
	mu    sync.Mutex
	rooms map[string]*roomEntry // eventID -> sala activa
}
//constructor recibe las interfaces que definimos osea los puertos
func NewChatService(
	broker ports.MessageBroker,
	tokens ports.TokenValidator,
	events ports.EventLookup,
	//usar el puntero para usar el mismo candando
) *ChatService {
	return &ChatService{
		broker: broker,
		tokens: tokens,
		events: events,
		//la lista en memoria que vive en el servidor
		rooms:  make(map[string]*roomEntry),
	}
}
//la importancia del broker sirve para hacer balanceo de cargas

// attach mete al cliente en la sala del evento.
// Si es el primero, crea la sala y enciende su suscripción al broker.
//con esta funcion ponemos alguien en la sala de un evento
//crea una lista a medida que vayan llegando usarios
func (s *ChatService) attach(eventID string, client *domain.Client) error {
	s.mu.Lock()
	//dajmos una nora para luego abrir el candado
	defer s.mu.Unlock()

	entry, ok := s.rooms[eventID]
	if !ok {
		// la sala no existe: la creamos con su propio ctx (no el de ningún usuario
		//creamos un base del creador donde los usaurios e van uniendo
		subCtx, cancel := context.WithCancel(context.Background())

		//sintonizar al evento por medio del broker
		//osea le pasamos los paremtros del vento y el contexto de la subscriçipn
		messages, err := s.broker.Subscribe(subCtx, eventID)
		if err != nil {
			cancel() // no quedó nada encendido, soltamos el ctx
			return err
		}

		entry = &roomEntry{room: domain.NewRoom(eventID), cancel: cancel}
		s.rooms[eventID] = entry

		go s.pump(entry.room, messages)
	}

	//agregar al cliente a la lista de la sala
	entry.room.AddClient(client)
	return nil
}

// pump toma cada mensaje que llega del broker y lo reparte a la sala
//empueja los mensjae hacia la sala
//solo los recibe nada mas osea leer
//lo que va hacer ese empujar los mensaje por toda la lista

func (s *ChatService) pump(room *domain.Room, messages <-chan domain.Message) {
	for msg := range messages {
		room.Broadcast(msg)
	}
}


// Join deja entrar a una persona al chat de un evento.
// token puede venir vacío: entra como anónimo y solo podrá leer.
func (s *ChatService) Join(ctx context.Context, eventID, token string) (*domain.Client, error) {
	// 1. ¿el evento existe y no ha terminado?
	//revisamos si el evento existe o termino
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event.Status == domain.EventFinished {
		return nil, ErrChatClosed
	}

	// 2. ¿quién es? Sin token es anónimo
	//revisa el token del usuario
	userID, username := "", ""
	if token != "" {
		user, err := s.tokens.Validate(token)
		if err != nil {
			return nil, err
		}
		userID, username = user.ID, user.Username
	}

	client := domain.NewClient(rand.Text(), userID, username)

	// 3. lo metemos en la lista de la sala
	//lo mete en la sala con attach y devuleve el cliente
	err = s.attach(eventID, client)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Send publica un mensaje de un cliente en el chat de su evento.
func (s *ChatService) Send(ctx context.Context, eventID string, client *domain.Client, text string) error {
	// 1. un anónimo no puede escribir
	if client.UserID == "" {
		//devolvemos la variable porque este no a sido autenticado
		return ErrNotAuthenticated
	}

	// 2. el domain crea el mensaje y valida (no vacío, máximo 200 caracteres)
	msg, err := domain.NewMessage(eventID, client.UserID, client.Username, text)
	if err != nil {
		return err
	}

	// 3. se lo pasamos al broker para que llegue a todas las réplicas
	return s.broker.Publish(ctx, msg)
}

// Leave saca al cliente de la sala. Si era el último, apaga la sala.
func (s *ChatService) Leave(eventID string, client *domain.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.rooms[eventID]
	if !ok {
		return
	}

	entry.room.RemoveClient(client.ID)

	if entry.room.Count() == 0 {
		entry.cancel()
		delete(s.rooms, eventID)
	}
}