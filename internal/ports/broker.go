package ports

import (
	"context"

	"horizon/flow/internal/domain"
)

// MessageBroker es el canal por donde viajan los mensajes entre todas las réplicas de Flow
type MessageBroker interface {
	// Publish envía un mensaje a todas las réplicas que escuchan la sala de ese evento
	//el ctx es la conveccion 
	Publish(ctx context.Context, msg domain.Message) error

	// Subscribe escucha los mensajes de la sala de un evento
	// el canal que devuelve se cierra cuando ctx se cancela
	//soloe s un canal de legura y refernciamos al messaje que esta en el apquete de domain
	//la primera en masyucula porque son publicos deben ser usas esas firmas
	//context.Context es la interafz inetrna que nos da Go
	Subscribe(ctx context.Context, eventID string) (<-chan domain.Message, error)
}