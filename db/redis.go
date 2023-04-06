package db

import (
	"sync"

	"time"

	"github.com/go-redis/redis"
)

var (
	redisService *Redis
	getRedisLock sync.Mutex
)

type Redis struct {
	client    *redis.Client
	keyPrefix string
}

func GetRedisService(addr string, password string, keyPrefix string, selectBb int) *Redis {
	if redisService == nil {
		getRedisLock.Lock()
		defer getRedisLock.Unlock()
		if redisService == nil {
			redisService = new(Redis)
			client := redis.NewClient(&redis.Options{
				Addr:     addr,
				Password: password,
				DB:       selectBb,
			})
			redisService.client = client
			redisService.keyPrefix = keyPrefix
		}
	}
	return redisService
}

// 从左边提取一条消息
func (redis *Redis) Close() error {
	return redis.client.Close()
}

// 从左边提取一条消息
func (redis *Redis) Keys(pattern string) *redis.StringSliceCmd {
	return redis.client.Keys(pattern)
}

// 从左边提取一条消息
func (redis *Redis) LPop(key string) *redis.StringCmd {
	return redis.client.LPop(redis.keyPrefix + key)
}

// 从右边提取一条消息
func (redis *Redis) RPop(key string) *redis.StringCmd {
	return redis.client.RPop(redis.keyPrefix + key)
}

// 发布一条消息
func (redis *Redis) LPush(key string, message interface{}) *redis.IntCmd {
	return redis.client.LPush(redis.keyPrefix+key, message)
}

// 发布一条消息
func (redis *Redis) RPush(key string, message interface{}) *redis.IntCmd {
	return redis.client.RPush(redis.keyPrefix+key, message)
}

// 批量设置过期时间
func (redis *Redis) BatchExpire(keyPattern string, second int) (interface{}, error) {
	return redis.client.Eval("local keys = redis.call('keys', ARGV[1]) for i=1,#keys,1 do redis.call('expire', keys[i], ARGV[2]) end return #keys",
		[]string{}, redis.keyPrefix+keyPattern, second).Result()
}

// 批量删除
func (redis *Redis) BatchDel(keyPattern string) (interface{}, error) {
	return redisService.client.Eval("local keys = redis.call('keys', ARGV[1]) for i=1,#keys,5000 do redis.call('del', unpack(keys, i, math.min(i+4999, #keys))) end return #keys",
		[]string{}, redis.keyPrefix+keyPattern).Result()
}

// 获取值
func (redis *Redis) Get(key string) *redis.StringCmd {
	return redis.client.Get(redis.keyPrefix + key)
}

// 设置值
func (redis *Redis) Set(key string, value interface{},
	expiration time.Duration) error {
	return redis.client.Set(redis.keyPrefix+key, value, expiration).Err()
}

// 递增
func (redis *Redis) Incr(key string) (int64, error) {
	return redis.client.Incr(redis.keyPrefix + key).Result()
}

// 递增步值
func (redis *Redis) IncrBy(key string, value int64) (int64, error) {
	return redis.client.IncrBy(redis.keyPrefix+key, value).Result()
}

// 递减
func (redis *Redis) Decr(key string) (int64, error) {
	return redis.client.Decr(redis.keyPrefix + key).Result()
}

// 递减
func (redis *Redis) DecrBy(key string, decrement int64) (int64, error) {
	return redis.client.DecrBy(redis.keyPrefix+key, decrement).Result()
}

// 哈希递增
func (redis *Redis) HIncrBy(key string, field string, incr int64) (int64, error) {
	return redis.client.HIncrBy(redis.keyPrefix+key, field, incr).Result()
}

// 哈希递增
func (redis *Redis) HMGet(key string, fields ...string) *redis.SliceCmd {
	return redis.client.HMGet(redis.keyPrefix+key, fields...)
}

// 获取所有哈希
func (redis *Redis) HGetAll(key string) *redis.StringStringMapCmd {
	return redis.client.HGetAll(redis.keyPrefix + key)
}

// 哈希设置
func (redis *Redis) HMSet(key string, fields map[string]interface{}) *redis.StatusCmd {
	return redis.client.HMSet(redis.keyPrefix+key, fields)
}

// 删除Key
func (redis *Redis) Del(key string) *redis.IntCmd {
	return redis.client.Del(redis.keyPrefix + key)
}

// 设置过期Key
func (redis *Redis) Expire(key string, expiration time.Duration) *redis.BoolCmd {
	return redis.client.Expire(redis.keyPrefix+key, expiration)
}
