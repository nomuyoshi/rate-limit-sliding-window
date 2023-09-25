package redis

import "github.com/redis/go-redis/v9"

func newClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}

var client *redis.Client

func SetClient() {
	if client == nil {
		client = newClient()
	}
}

func GetClient() *redis.Client {
	if client == nil {
		panic("should call SetClient")
	}
	return client
}
