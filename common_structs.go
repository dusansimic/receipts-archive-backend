package main

import (
	"github.com/gbrlsnchs/jwt/v3"
)

// JWTPayload : Structure that is only used for JWT Payload
type JWTPayload struct {
	jwt.Payload
	UserID string `json:"id"`
}

// ContextKey is a custom type string for context key
type ContextKey string

// StructID : Structure for getting id
type StructID struct {
	ID int `db:"id"`
}
