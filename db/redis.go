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
	Client    *redis.Client
	KeyPrefix string
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
			redisService.Client = client
			redisService.KeyPrefix = keyPrefix
		}
	}
	return redisService
}

// 从左边提取一条消息
func (redis *Redis) Close() error {
	return redis.Client.Close()
}

// 从左边提取一条消息
func (redis *Redis) Keys(pattern string) *redis.StringSliceCmd {
	return redis.Client.Keys(pattern)
}

// 从左边提取一条消息
func (redis *Redis) LPop(key string) *redis.StringCmd {
	return redis.Client.LPop(redis.KeyPrefix + key)
}

// 从右边提取一条消息
func (redis *Redis) RPop(key string) *redis.StringCmd {
	return redis.Client.RPop(redis.KeyPrefix + key)
}

// 发布一条消息
func (redis *Redis) LPush(key string, message interface{}) *redis.IntCmd {
	return redis.Client.LPush(redis.KeyPrefix+key, message)
}

// 发布一条消息
func (redis *Redis) RPush(key string, message interface{}) *redis.IntCmd {
	return redis.Client.RPush(redis.KeyPrefix+key, message)
}

// 批量设置过期时间
func (redis *Redis) BatchExpire(keyPattern string, second int) (interface{}, error) {
	return redis.Client.Eval("local keys = redis.call('keys', ARGV[1]) for i=1,#keys,1 do redis.call('expire', keys[i], ARGV[2]) end return #keys",
		[]string{}, redis.KeyPrefix+keyPattern, second).Result()
}

// 批量删除
func (redis *Redis) BatchDel(keyPattern string) (interface{}, error) {
	return redisService.Client.Eval("local keys = redis.call('keys', ARGV[1]) for i=1,#keys,5000 do redis.call('del', unpack(keys, i, math.min(i+4999, #keys))) end return #keys",
		[]string{}, redis.KeyPrefix+keyPattern).Result()
}

// 获取值
func (redis *Redis) Get(key string) *redis.StringCmd {
	return redis.Client.Get(redis.KeyPrefix + key)
}

// 设置值
func (redis *Redis) Set(key string, value interface{},
	expiration time.Duration) error {
	return redis.Client.Set(redis.KeyPrefix+key, value, expiration).Err()
}

// 递增
func (redis *Redis) Incr(key string) (int64, error) {
	return redis.Client.Incr(redis.KeyPrefix + key).Result()
}

// 递增步值
func (redis *Redis) IncrBy(key string, value int64) (int64, error) {
	return redis.Client.IncrBy(redis.KeyPrefix+key, value).Result()
}

// 递减
func (redis *Redis) Decr(key string) (int64, error) {
	return redis.Client.Decr(redis.KeyPrefix + key).Result()
}

// 递减
func (redis *Redis) DecrBy(key string, decrement int64) (int64, error) {
	return redis.Client.DecrBy(redis.KeyPrefix+key, decrement).Result()
}

// 哈希递增
func (redis *Redis) HIncrBy(key string, field string, incr int64) (int64, error) {
	return redis.Client.HIncrBy(redis.KeyPrefix+key, field, incr).Result()
}

// 哈希递增
func (redis *Redis) HMGet(key string, fields ...string) *redis.SliceCmd {
	return redis.Client.HMGet(redis.KeyPrefix+key, fields...)
}

// 获取所有哈希
func (redis *Redis) HGetAll(key string) *redis.StringStringMapCmd {
	return redis.Client.HGetAll(redis.KeyPrefix + key)
}

// 哈希设置
func (redis *Redis) HMSet(key string, fields map[string]interface{}) *redis.StatusCmd {
	return redis.Client.HMSet(redis.KeyPrefix+key, fields)
}

// 删除Key
func (redis *Redis) Del(key string) *redis.IntCmd {
	return redis.Client.Del(redis.KeyPrefix + key)
}

// 设置过期Key
func (redis *Redis) Expire(key string, expiration time.Duration) *redis.BoolCmd {
	return redis.Client.Expire(redis.KeyPrefix+key, expiration)
}
