package jwtvalidator

import (
	"crypto/rsa"
	"os"

	"github.com/golang-jwt/jwt/v5"

	"horizon/flow/internal/domain"
	"horizon/flow/internal/ports"
	"log"
)

// comprobación al compilar: Validator debe cumplir el port TokenValidator
var _ ports.TokenValidator = (*Validator)(nil)

// Validator verifica los tokens que firma Identity, usando su clave pública
type Validator struct {
	publicKey *rsa.PublicKey
	parser    *jwt.Parser
}

// claims es lo que esperamos encontrar dentro del token
type claims struct {
	jwt.RegisteredClaims
	Role string `json:"http://schemas.microsoft.com/ws/2008/06/identity/claims/role"`
}

// New carga la clave pública desde un archivo .pem
func New(publicKeyPath string) (*Validator, error) {
	pemBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM(pemBytes)
	if err != nil {
		return nil, err
	}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer("Horizon.Identity"),
		jwt.WithAudience("Horizon.Platform"),
		jwt.WithExpirationRequired(),
	)

	return &Validator{publicKey: key, parser: parser}, nil
}

func (v *Validator) Validate(tokenString string) (domain.User, error) {
	var c claims

	token, err := v.parser.ParseWithClaims(tokenString, &c, func(t *jwt.Token) (any, error) {
		return v.publicKey, nil
	})
	if err != nil || !token.Valid || c.Subject == "" {
		log.Printf("token rechazado: %v", err)
		return domain.User{}, ports.ErrInvalidToken
	}

	return domain.User{
		ID:       c.Subject,
		Username: usernameFor(c.Subject, c.Role),
		Role:     c.Role,
	}, nil
}

// usernameFor arma un nombre provisional, porque el token no trae uno.
// Ejemplo: "Creador-312b". No usa el email, que es privado.
func usernameFor(sub, role string) string {
	if role == "" {
		role = "Usuario"
	}

	short := sub
	if len(short) > 4 {
		short = short[:4]
	}

	return role + "-" + short
}