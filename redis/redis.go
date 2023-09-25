package redis

import "github.com/redis/go-redis/v9"

func newClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}

var client *redis.Client

func Client() *redis.Client {
	if client == nil {
		client = newClient()
	}

	return client
}
