package db

import (
	"errors"
	"time"

	"github.com/go-redis/redis"
	"github.com/ishanjain28/envelope-backend/common"
)

// Redis Client in DB has context set from Handle function in Router

// VerifyDeviceID takes a Device ID and checks if it registered via checking it's existence in Redis.
func (d *DB) VerifyDeviceID(deviceid string, hash string) error {

	h, err := d.Redis.Get(deviceid).Result()
	if err != nil {

		if err == redis.Nil {
			return errors.New(common.ErrNotRegistered)
		}
		return err
	}

	if h != hash {
		return errors.New(common.ErrInvalidData)
	}

	return nil
}

func (d *DB) RegisterDeviceID(deviceid string, hash string, t time.Duration) error {
	return d.Redis.Set(deviceid, hash, t).Err()
}
