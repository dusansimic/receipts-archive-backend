package database

import "github.com/bradfitz/gomemcache/memcache"

// MemcachedOptions options for memcached database
type MemcachedOptions struct {
	Addr string
}

// NewConnection creates a new connection to memcached datgabase
func (o MemcachedOptions) NewConnection() *memcache.Client {
	return memcache.New(o.Addr)
}
