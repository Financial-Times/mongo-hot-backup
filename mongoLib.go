package main

import (
	"gopkg.in/mgo.v2"
)

type mongoLib interface {
	Dial(url string) (mongoSession, error)
}

type labixMongo struct {}

func (m *labixMongo) Dial(url string) (*labixSession, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}
	return &labixSession{session}, nil
}

type mongoSession interface {
	SetPrefetch(p float64)
	Close()
	DB(name string) mongoDatabase
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

func (s *labixSession) Snapshot(database, collection string, findQuery interface{}) *labixQuery {
	return &labixQuery{s.session.DB(database).C(collection).Find(findQuery).Snapshot()}
}

func (s *labixSession) DB(name string) *labixDatabase {
	return &labixDatabase{s.session.DB(name)}
}

type mongoDatabase interface {
	C(name string) mongoCollection
}

type labixDatabase struct {
	database *mgo.Database
}

func (d *labixDatabase) C(name string) *labixCollection {
	return &labixCollection{d.database.C(name)}
}

type mongoCollection interface {
	Find(query interface{}) mongoQuery
}

type labixCollection struct {
	collection *mgo.Collection
}

func (c *labixCollection) Find(query interface{}) *labixQuery {
	return &labixQuery{c.collection.Find(query)}
}

type mongoQuery interface {
	Snapshot() mongoQuery
	Iter() mongoIter
}

type labixQuery struct {
	query *mgo.Query
}

func (q *labixQuery) Snapshot() *labixQuery {
	return &labixQuery{q.query.Snapshot()}
}

func (q *labixQuery) Iter() *labixIter {
	return &labixIter{q.query.Iter()}
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
