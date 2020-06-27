package handlers

import (
	"github.com/go-playground/validator"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
)

// ContextKey is a custom type string for context key
type ContextKey string

// StructID : Structure for getting id
type StructID struct {
	ID int `db:"id"`
}

// Options stores database and validator options for handlers
type Options struct {
	DB  *sqlx.DB
	RDB *redis.Client
	V   *validator.Validate
}
