package domain

import "sync"

// Room es la sala de chat de un evento: sabe quiénes están conectados y les reparte los mensajes
type Room struct {
	ID string // es el id del evento

	mu      sync.RWMutex       // candado que protege a clients
	//Creamos un diccionario en la memoria del servidor
	//es minusucla en clients ya que es un mapa privado nadie puede acceders
	clients map[string]*Client // clientes conectados, por id de conexión
}

// NewRoom crea una sala vacía
func NewRoom(id string) *Room {
	return &Room{
		ID:      id,
		clients: make(map[string]*Client),
	}
}

// AddClient mete un cliente a la sala
func (r *Room) AddClient(c *Client) {
	//modicar
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clients[c.ID] = c
}

// RemoveClient saca un cliente de la sala (por ejemplo cuando se desconecta)
func (r *Room) RemoveClient(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.clients, clientID)
}

// Count dice cuántos clientes hay conectados
func (r *Room) Count() int {
	//leer
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.clients)
}

// Broadcast reparte el mensaje a todos los clientes de la sala
// devuelve los clientes que no pudieron recibirlo (su cola estaba llena) y que ya fueron sacados
func (r *Room) Broadcast(msg Message) []*Client {
	var slow []*Client

	r.mu.RLock()
	for _, c := range r.clients {
		if !c.Send(msg) {
			slow = append(slow, c)
		}
	}
	r.mu.RUnlock()

	// los lentos se sacan de la sala y se les avisa que deben desconectarse
	for _, c := range slow {
		r.RemoveClient(c.ID)
		c.Close()
	}

	return slow
}