package thisapp

import (
	"time"

	"github.com/garyburd/redigo/redis"
)

type RedisConnection interface {
	GetConn() redis.Conn
}

type MyRedis struct {
	pool *redis.Pool
}

func NewRedisConnection(server string, maxIdle, maxActive int, idleTimeout int64) RedisConnection {
	return &MyRedis{
		pool: &redis.Pool{
			MaxIdle:     maxIdle,
			MaxActive:   maxActive,
			IdleTimeout: time.Duration(idleTimeout) * time.Second,
			Wait:        true,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", server)
				if err != nil {
					return nil, err
				}
				return c, err
			},
		},
	}
}

func (r *MyRedis) GetConn() redis.Conn {
	return r.pool.Get()
}