package domain

// EventStatus es el estado del evento, igual que en el servicio Events
type EventStatus int

const (
	//hacer enum con iota 
	EventScheduled EventStatus = iota // 0
	EventLive                         // 1
	EventFinished                     // 2
)

// EventInfo es lo mínimo que Flow necesita saber de un evento
type EventInfo struct {
	ID     string
	Status EventStatus
}