package aorm

import "time"

func (s *DB) Deadline() (deadline time.Time, ok bool) {
	return s.Context.Deadline()
}

func (s *DB) Done() <-chan struct{} {
	return s.Context.Done()
}

func (s *DB) Err() error {
	if s.Error == nil {
		return s.Context.Err()
	}
	return s.Error
}

func (s *DB) Value(key interface{}) interface{} {
	if str, ok := key.(string); ok {
		if v, ok := s.values[str]; ok {
			return v
		}
	}
	return s.Context.Value(key)
}
