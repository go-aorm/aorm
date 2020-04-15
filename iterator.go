package aorm

import "github.com/pkg/errors"

type IteratorHeader struct {
	Value interface{}
}

type RecordsIterator interface {
	Start() (state interface{}, err error)
	Done(state interface{}) (ok bool)
	Next(state interface{}) (recorde, newState interface{}, err error)
}

func Iterate(it RecordsIterator) (err error) {
	var state interface{}
	for state, err = it.Start(); err == nil && !it.Done(state); _, state, err = it.Next(state) {
	}
	return
}

type RecordsIteratorOpener interface {
	Open() (header interface{}, it RecordsIterator, err error)
}

func OpenIterate(opener RecordsIteratorOpener) (err error) {
	var it RecordsIterator
	if _, it, err = opener.Open(); err != nil {
		return errors.Wrap(err, "OpenIterate > Open() failed")
	}
	return Iterate(it)
}

func Each(it RecordsIterator, cb func(i int, r interface{}) error) (err error) {
	var (
		state, rec interface{}
		i          int
	)
	for state, err = it.Start(); err == nil && !it.Done(state); i++ {
		if rec, state, err = it.Next(state); err == nil {
			err = cb(i, rec)
		}
	}
	return
}
