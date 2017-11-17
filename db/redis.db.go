package db

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis"
)

// Redis Client in DB has context set from Handle function in Router

// VerifyDeviceID takes a Device ID and checks if it registered via checking it's existence in Redis.
func (d *DB) VerifyDeviceID(ctx context.Context, deviceid string) (string, error) {

	h, err := d.Redis.Get(deviceid).Result()
	if err != nil {

		if err == redis.Nil {
			return "", errors.New(ErrNotRegistered)
		}
		return "", err
	}

	return h, nil
}

// RegisterDeviceID takes a device id and a hash and saves it in database
func (d *DB) RegisterDeviceID(ctx context.Context, deviceid string, hash string, t time.Duration) error {
	return d.Redis.Set(deviceid, hash, t).Err()
}
