package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"pc4/tools"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var redisDB *redis.Client

func InitRedis() error {
	addr, err := os.LookupEnv("REDIS_ADDR")
	if !err {
		return fmt.Errorf("REDIS_ADDR not set")
	}

	rdb := redis.NewClient(&redis.Options{Addr: addr})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect redis at %s: %w", addr, err)
	}

	redisDB = rdb
	return nil
}

func GetRedisDB() *redis.Client {
	return redisDB
}

func SaveToRedis(rdb *redis.Client, allUsers map[int]tools.UserVector) {
	for userID, vec := range allUsers {
		jsonBytes, _ := json.Marshal(vec)
		key := fmt.Sprintf("user:%d", userID)
		rdb.Set(ctx, key, jsonBytes, 0)
	}
}

func LoadFromRedis(rdb *redis.Client) map[int]tools.UserVector {
	result := make(map[int]tools.UserVector)

	keys, _ := rdb.Keys(ctx, "user:*").Result()

	for _, key := range keys {
		data, _ := rdb.Get(ctx, key).Result()

		var vec tools.UserVector
		json.Unmarshal([]byte(data), &vec)

		var uid int
		fmt.Sscanf(key, "user:%d", &uid)

		result[uid] = vec
	}
	return result
}
