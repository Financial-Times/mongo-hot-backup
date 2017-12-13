package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
)

type statusKeeper interface {
	Save(result backupResult) error
	Get(coll fullColl) (backupResult, error)
}

type boltStatusKeeper struct {
	db *bolt.DB
}

func newBoltStatusKeeper(dbPath string) (*boltStatusKeeper, error) {
	err := os.MkdirAll(filepath.Dir(dbPath), 0600)

	if err != nil {
		return nil, err
	}
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Results"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &boltStatusKeeper{db}, nil
}

func (s *boltStatusKeeper) Save(result backupResult) error {
	r, _ := json.Marshal(result)
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Results"))
		err := b.Put([]byte(fmt.Sprintf("%s/%s", result.Collection.database, result.Collection.collection)), r)
		return err
	})
}

func (s *boltStatusKeeper) Get(coll fullColl) (backupResult, error) {
	var result backupResult
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Results"))
		v := b.Get([]byte(fmt.Sprintf("%s/%s", coll.database, coll.collection)))

		err := json.Unmarshal(v, &result)
		if err != nil {
			return err
		}

		return nil
	})
	return result, err
}
