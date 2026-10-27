package cache

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"roadmates/backend/internal/service"
)

type Redis struct{ client *redis.Client }

func NewRedis(client *redis.Client) *Redis { return &Redis{client} }

const operationTimeout = 150 * time.Millisecond

func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	data, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, service.ErrCacheMiss
	}
	return data, err
}
func (r *Redis) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return r.client.Set(ctx, key, data, ttl).Err()
}
func (r *Redis) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return r.client.Del(ctx, key).Err()
}

// A random starting counter prevents reuse of old keys if allkeys-lru evicts
// just the namespace counter while some search entries remain in Redis.
var versionScript = redis.NewScript(`
if redis.call('EXISTS',KEYS[1]) == 0 then redis.call('SET',KEYS[1],ARGV[1]) end
if ARGV[2] == 'bump' then redis.call('INCR',KEYS[1]) end
return redis.call('GET',KEYS[1])`)

func (r *Redis) version(ctx context.Context, namespace, action string) (string, error) {
	var seed [8]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return "", err
	}
	initial := strconv.FormatUint((binary.BigEndian.Uint64(seed[:])&((1<<52)-1))+1, 10)
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return versionScript.Run(ctx, r.client, []string{"roadmates:v1:" + namespace + ":version"}, initial, action).Text()
}
func (r *Redis) NamespaceVersion(ctx context.Context, namespace string) (string, error) {
	return r.version(ctx, namespace, "read")
}
func (r *Redis) BumpNamespace(ctx context.Context, namespace string) error {
	_, err := r.version(ctx, namespace, "bump")
	return err
}
