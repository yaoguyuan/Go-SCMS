package utils

import (
	"auth/initializers"
	"time"
)

const (
	LOGIN_CODE_KEY_PREFIX  = "login:code:"
	LOGIN_CODE_EXPIRE_TIME = 5 * time.Minute
	CACHE_USER_KEY_PREFIX  = "cache:user:"
	CACHE_USER_EXPIRE_TIME = 30 * time.Minute
	CACHE_NULL_EXPIRE_TIME = 2 * time.Minute
	MUTEX_USER_KEY_PREFIX  = "mutex:user:"
	MUTEX_USER_EXPIRE_TIME = 1 * time.Second
)

func TryLock(key string, ttl time.Duration) bool {
	return initializers.RDB.SetNX(initializers.RDB_CTX, key, "1", ttl).Val()
}

func Unlock(key string) {
	initializers.RDB.Del(initializers.RDB_CTX, key)
}
