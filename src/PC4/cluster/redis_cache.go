package cluster

import (
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func saveToRedis(rdb *redis.Client, allUsers map[int]UserVector) {
	for userID, vec := range allUsers {
		jsonBytes, _ := json.Marshal(vec)
		key := fmt.Sprintf("user:%d", userID)
		rdb.Set(ctx, key, jsonBytes, 0)
	}
}

func loadFromRedis(rdb *redis.Client) map[int]UserVector {
	result := make(map[int]UserVector)

	keys, _ := rdb.Keys(ctx, "user:*").Result()

	for _, key := range keys {
		data, _ := rdb.Get(ctx, key).Result()

		var vec UserVector
		json.Unmarshal([]byte(data), &vec)

		var uid int
		fmt.Sscanf(key, "user:%d", &uid)

		result[uid] = vec
	}
	return result
}
