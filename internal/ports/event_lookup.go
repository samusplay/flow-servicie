package ports

import (
	"context"
	"errors"

	"horizon/flow/internal/domain"
)

// ErrEventNotFound se devuelve cuando el evento no existe
var ErrEventNotFound = errors.New("el evento no existe")

// EventLookup consulta a quien sea dueño de los eventos (hoy, el servicio Events)
//contrati para buscar al eventoe ste viene de afuera de events
type EventLookup interface {
	//guardamos lsolo informacion no nos intersa el alamcenamiento de todo el evento
	//con esta firma le preguntamos a events por Red
	//context.context ayuda manjera el ciclod e vida
	//sirve par aorgramar señales de cancelacion
	//ponerle un limite 
	Get(ctx context.Context, eventID string) (domain.EventInfo, error)
}