package cluster

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func connectRedis() *redis.Client {
	addr, ok := os.LookupEnv("REDIS_ADDR")
	if !ok {
		fmt.Errorf("REDIS_ADDR not set")
	}

	return redis.NewClient(&redis.Options{
		Addr: addr,
	})
}
