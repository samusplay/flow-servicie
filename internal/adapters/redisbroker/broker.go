package redisbroker

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"

	"horizon/flow/internal/domain"
	"horizon/flow/internal/ports"
)

// Broker habla con Redis para repartir los mensajes entre réplicas
//guarda el objeto con la cofniguracion de Redis
type Broker struct {
	client *redis.Client
}
// comprobación al compilar: Broker debe cumplir el port MessageBroker
var _ ports.MessageBroker = (*Broker)(nil)

// New crea el broker apuntando a la dirección de Redis, por ejemplo "localhost:6379"
func New(addr string) *Broker {
	return &Broker{
		client: redis.NewClient(&redis.Options{Addr: addr}),
	}
}

// channelName es el canal de Redis donde viven los mensajes de un evento
//arma el canal del evento
func channelName(eventID string) string {
	return "chat:" + eventID
}

// Publish manda el mensaje a todas las réplicas que escuchan ese evento
//implemntemos el port que habiamos definido le pasamos los parametros
func (b *Broker) Publish(ctx context.Context, msg domain.Message) error {
	//convierte el mensaje a Json
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return b.client.Publish(ctx, channelName(msg.EventID), data).Err()
}

// Subscribe escucha el canal del evento y entrega cada mensaje por un channel de Go
func (b *Broker) Subscribe(ctx context.Context, eventID string) (<-chan domain.Message, error) {
	sub := b.client.Subscribe(ctx, channelName(eventID))

	// esperamos a que Redis confirme que la suscripción quedó activa
	_, err := sub.Receive(ctx)
	if err != nil {
		sub.Close()
		return nil, err
	}

	out := make(chan domain.Message, 64)

	go func() {
		defer close(out)
		defer sub.Close()

		redisMessages := sub.Channel()

		for {
			select {
			case <-ctx.Done():
				return

			case rm, ok := <-redisMessages:
				if !ok {
					return
				}

				var msg domain.Message
				err := json.Unmarshal([]byte(rm.Payload), &msg)
				if err != nil {
					continue
				}

				select {
				case out <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}