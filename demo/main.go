package main

import (
	"log"
	"os"

	"horizon/flow/internal/adapters/jwtvalidator"
)

func main() {
	v, err := jwtvalidator.New("../Horizon.Events/Security/Keys/public.pem")
	if err != nil {
		log.Fatal(err)
	}

	user, err := v.Validate(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	log.Println("usuario:", user.ID, user.Username, user.Role)
}