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
	Get(coll dbColl) (backupResult, error)
}

type boltStatusKeeper struct {
	dbPath string
}

func newBoltStatusKeeper(dbPath string) *boltStatusKeeper {
	return &boltStatusKeeper{dbPath}
}

func (s *boltStatusKeeper) Save(result backupResult) error {
	err := os.MkdirAll(filepath.Dir(s.dbPath), 0600)
	if err != nil {
		return err
	}
	db, err := bolt.Open(s.dbPath, 0600, nil)
	if err != nil {
		return err
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
		return err
	}

	r, _ := json.Marshal(result)
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Results"))
		err := b.Put([]byte(fmt.Sprintf("%s/%s", result.Collection.database, result.Collection.collection)), r)
		return err
	})
}

func (s *boltStatusKeeper) Get(coll dbColl) (backupResult, error) {
	db, err := bolt.Open(s.dbPath, 0600, nil)
	if err != nil {
		return backupResult{}, err
	}
	defer db.Close()

	var result backupResult
	err = db.View(func(tx *bolt.Tx) error {
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
