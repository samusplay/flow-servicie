package domain

const (
	RoleCreator   = "Creador"
	RoleSpectator = "Espectador"
)

// User es una persona ya identificada que usa el chat
type User struct {
	ID       string
	Username string
	Role     string
}