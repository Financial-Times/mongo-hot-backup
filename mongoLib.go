package main

import (
	"gopkg.in/mgo.v2"
)

type mongoLib interface {
	Dial(url string) (mongoSession, error)
}

type labixMongo struct {}

func (m *labixMongo) Dial(url string) (mongoSession, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}
	return &labixSession{session}, nil
}

type mongoSession interface {
	SetPrefetch(p float64)
	Close()
	SnapshotIter(database, collection string, findQuery interface{}) mongoIter
}

type labixSession struct {
	session *mgo.Session
}

func (s *labixSession) SetPrefetch(p float64) {
	s.session.SetPrefetch(p)
}

func (s *labixSession) Close() {
	s.session.Close()
}

func (s *labixSession) SnapshotIter(database, collection string, findQuery interface{}) mongoIter {
	return &labixIter{s.session.DB(database).C(collection).Find(findQuery).Snapshot().Iter()}
}

type mongoIter interface {
	Next(result interface{}) bool
	Err() error
}

type labixIter struct {
	iter *mgo.Iter
}

func (i *labixIter) Next(result interface{}) bool {
	return i.iter.Next(result)
}

func (i *labixIter) Err() error {
	return i.Err()
}
