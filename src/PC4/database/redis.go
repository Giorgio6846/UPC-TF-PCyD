package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"pc4/tools"
	"strconv"
	"strings"
	"sync"

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

func LoadFromRedis(ctx context.Context, rdb *redis.Client) (map[int]tools.UserVector, error) {
	const (
		scanCount = 500
		batchSize = 200
		workers   = 8
	)

	out := make(map[int]tools.UserVector)
	var mu sync.Mutex

	keysCh := make(chan string, scanCount*2)
	errCh := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()

			batch := make([]string, 0, batchSize)

			flush := func() error {
				if len(batch) == 0 {
					return nil
				}
				pipe := rdb.Pipeline()
				cmds := make([]*redis.StringCmd, len(batch))
				for i, k := range batch {
					cmds[i] = pipe.Get(ctx, k)
				}
				_, err := pipe.Exec(ctx)
				if err != nil && err != redis.Nil {
					return err
				}

				for i, k := range batch {
					raw, err := cmds[i].Result()
					if err == redis.Nil {
						continue
					}
					if err != nil {
						return err
					}

					var vec tools.UserVector
					if err := json.Unmarshal([]byte(raw), &vec); err != nil {
						return err
					}

					uidStr := strings.TrimPrefix(k, "user:")
					uid, err := strconv.Atoi(uidStr)
					if err != nil {
						return err
					}

					mu.Lock()
					out[uid] = vec
					mu.Unlock()
				}

				batch = batch[:0]
				return nil
			}

			for k := range keysCh {
				batch = append(batch, k)
				if len(batch) >= batchSize {
					if err := flush(); err != nil {
						select {
						case errCh <- err:
						default:
						}
						return
					}
				}
			}
			// flush remainder
			if err := flush(); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
		}()
	}

	var cursor uint64
	for {
		keys, next, err := rdb.Scan(ctx, cursor, "user:*", scanCount).Result()
		if err != nil {
			close(keysCh)
			wg.Wait()
			return nil, err
		}
		cursor = next

		for _, k := range keys {
			keysCh <- k
		}

		if cursor == 0 {
			break
		}
	}

	close(keysCh)
	wg.Wait()

	select {
	case err := <-errCh:
		return nil, err
	default:
		return out, nil
	}
}
