package ports

import (
	"errors"

	"horizon/flow/internal/domain"
)

// ErrInvalidToken se devuelve cuando el token es falso, está mal formado o expiró
var ErrInvalidToken = errors.New("token inválido o expirado")

// TokenValidator verifica un JWT y dice quién es el usuario
type TokenValidator interface {
	Validate(token string) (domain.User, error)
}