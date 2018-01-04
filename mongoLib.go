package main

import (
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type mongoLib interface {
	DialWithTimeout(url string, timeout time.Duration) (mongoSession, error)
}

type labixMongo struct{}

func (m *labixMongo) DialWithTimeout(url string, timeout time.Duration) (mongoSession, error) {
	session, err := mgo.DialWithTimeout(url, timeout)
	if err != nil {
		return nil, err
	}
	session.SetSyncTimeout(timeout)
	session.SetSocketTimeout(timeout)
	session.SetPrefetch(1.0)
	return &labixSession{session}, nil
}

type mongoSession interface {
	Close()
	SnapshotIter(database, collection string, findQuery interface{}) mongoIter
	RemoveAll(database, collection string, removeQuery interface{}) error
	Bulk(database, collection string) mongoBulk
}

type labixSession struct {
	session *mgo.Session
}

func (s *labixSession) Close() {
	s.session.Close()
}

func (s *labixSession) SnapshotIter(database, collection string, findQuery interface{}) mongoIter {
	return &labixIter{s.session.DB(database).C(collection).Find(findQuery).Snapshot().Iter()}
}

func (s *labixSession) RemoveAll(database, collection string, removeQuery interface{}) error {
	_, err := s.session.DB(database).C(collection).RemoveAll(nil)
	return err
}

func (s *labixSession) Bulk(database, collection string) mongoBulk {
	return &labixBulk{s.session.DB(database).C(collection).Bulk()}
}

type mongoBulk interface {
	Run() error
	Insert(data []byte)
}

type labixBulk struct {
	bulk *mgo.Bulk
}

func (b *labixBulk) Run() error {
	_, err := b.bulk.Run()
	return err
}

func (b *labixBulk) Insert(data []byte) {
	b.bulk.Insert(bson.Raw{Data: data})
}

type mongoIter interface {
	Next() ([]byte, bool)
	Err() error
}

type labixIter struct {
	iter *mgo.Iter
}

func (i *labixIter) Next() ([]byte, bool) {
	result := &bson.Raw{}
	hasNext := i.iter.Next(result)
	return result.Data, hasNext
}

func (i *labixIter) Err() error {
	return i.iter.Err()
}
