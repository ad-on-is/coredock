package internal

import (
	bolt "go.etcd.io/bbolt"
)

type DB struct {
	db *bolt.DB
}

func NewDB() *DB {

	db, err := bolt.Open("data.db", 0600, nil)
	if err != nil {
		return nil
	}
	return &DB{db: db}
}

func (d *DB) Get(key string) string {
	var value []byte
	d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("services"))
		if b == nil {
			return nil
		}
		value = b.Get([]byte(key))
		return nil
	})
	v := string(value)
	logger.Debugf("DB: Found key %s with value %s", key, v)
	return v
}

func (d *DB) Set(key string, value string) {
	d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("services"))
		if err != nil {
			return err
		}
		logger.Debugf("DB: Setting key %s to value %s", key, value)
		return b.Put([]byte(key), []byte(value))
	})
}
