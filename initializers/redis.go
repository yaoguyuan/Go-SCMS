package initializers

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client
var RDB_CTX context.Context

func ConnectToRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASS"),
		DB:       0,
	})

	RDB_CTX = context.Background()

	// Test the connection
	if err := RDB.Ping(RDB_CTX).Err(); err != nil {
		panic("Failed to connect to Redis: " + err.Error())
	}
}
