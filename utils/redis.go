package utils

import (
	"auth/initializers"
	"os"
	"path/filepath"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConstants contains constants for Redis keys and expiration times.
var RedisConstants = struct {
	LOGIN_CODE_KEY_PREFIX  string
	LOGIN_CODE_EXPIRE_TIME time.Duration
	CACHE_USER_KEY_PREFIX  string
	CACHE_USER_EXPIRE_TIME time.Duration
	CACHE_NULL_EXPIRE_TIME time.Duration
	MUTEX_USER_KEY_PREFIX  string
	MUTEX_USER_EXPIRE_TIME time.Duration
	// LOCK_ORDER_KEY_PREFIX    string
	// LOCK_ORDER_EXPIRE_TIME   time.Duration
	SECKILL_STOCK_KEY_PREFIX string
	SECKILL_ORDER_KEY_PREFIX string
}{
	LOGIN_CODE_KEY_PREFIX:  "login:code:",
	LOGIN_CODE_EXPIRE_TIME: 5 * time.Minute,
	CACHE_USER_KEY_PREFIX:  "cache:user:",
	CACHE_USER_EXPIRE_TIME: 30 * time.Minute,
	CACHE_NULL_EXPIRE_TIME: 2 * time.Minute,
	MUTEX_USER_KEY_PREFIX:  "mutex:user:",
	MUTEX_USER_EXPIRE_TIME: 1 * time.Second,
	// LOCK_ORDER_KEY_PREFIX:    "lock:order:",
	// LOCK_ORDER_EXPIRE_TIME:   5 * time.Second,
	SECKILL_STOCK_KEY_PREFIX: "seckill:stock:",
	SECKILL_ORDER_KEY_PREFIX: "seckill:order:",
}

// SeckillScript is a Lua script used for atomic seckill operations in Redis
var SeckillScript *redis.Script

func init() {
	// Initialize the SeckillScript from ./scripts/seckill.lua
	workingDir, err := os.Getwd()
	if err != nil {
		panic("Failed to get working directory: " + err.Error())
	}
	seckillScriptPath := filepath.Join(workingDir, "utils", "scripts", "seckill.lua")
	seckillScriptContent, err := os.ReadFile(seckillScriptPath)
	if err != nil {
		panic("Failed to read seckill script: " + err.Error())
	}
	SeckillScript = redis.NewScript(string(seckillScriptContent))
}

func SimpleTryLock(key string, ttl time.Duration) bool {
	return initializers.RDB.SetNX(initializers.RDB_CTX, key, "1", ttl).Val()
}

func SimpleUnlock(key string) {
	initializers.RDB.Del(initializers.RDB_CTX, key)
}

// // unlockScript is a Lua script used for atomic unlocking in Redis
// var unlockScript = redis.NewScript(`
// 	if redis.call("get", KEYS[1]) == ARGV[1] then
// 		return redis.call("del", KEYS[1])
// 	end
// 	return 0
// `)
//
// // TryLock attempts to acquire a lock with a key and a time-to-live (ttl).
// // 'rid' is the uuid of the request, used to identify the lock owner
// func TryLock(key string, rid string, ttl time.Duration) bool {
// 	return initializers.RDB.SetNX(initializers.RDB_CTX, key, rid, ttl).Val()
// }
//
// // Unlock attempts to release a lock with a key.
// // 'rid' is the uuid of the request, used to identify the lock owner
// // It will only release the lock if the lock owner matches the provided 'rid'.
// func Unlock(key string, rid string) bool {
// 	// Using Lua script to ensure atomicity
// 	keys := []string{key}
// 	result, _ := unlockScript.Run(initializers.RDB_CTX, initializers.RDB, keys, rid).Int()
// 	return result == 1
// }
