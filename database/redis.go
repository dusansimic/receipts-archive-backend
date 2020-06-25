package database

import "github.com/go-redis/redis/v8"

// RedisOptions options for redis database
type RedisOptions struct {
	Addr string
	Password string
	DB int
}

// NewConnection creates a new connection to redis database
func (o RedisOptions) NewConnection() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: o.Addr,
		Password: o.Password,
		DB: o.DB,
	})
}
