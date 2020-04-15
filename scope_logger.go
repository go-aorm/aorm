package aorm

const (
	LOG_CREATE = "create"
	LOG_READ   = "read"
	LOG_UPDATE = "update"
	LOG_DELETE = "delete"
	LOG_QUERY  = "query"
	LOG_EXEC   = "exec"
)

var DefaultLogger = &ScopeLoggers{}

type ScopeLoggers struct {
	Async     bool
	callbacks map[string][]func(action string, scope *Scope)
}

func (s *ScopeLoggers) Register(action string, callback func(action string, scope *Scope)) *ScopeLoggers {
	if s.callbacks == nil {
		s.callbacks = map[string][]func(action string, scope *Scope){}
	}
	if _, ok := s.callbacks[action]; !ok {
		s.callbacks[action] = []func(action string, scope *Scope){callback}
	} else {
		s.callbacks[action] = append(s.callbacks[action], callback)
	}
	return s
}

func (s *ScopeLoggers) Update(callback func(action string, scope *Scope)) *ScopeLoggers {
	return s.Register(LOG_UPDATE, callback)
}

func (s *ScopeLoggers) Create(callback func(action string, scope *Scope)) *ScopeLoggers {
	return s.Register(LOG_CREATE, callback)
}

func (s *ScopeLoggers) Delete(callback func(action string, scope *Scope)) *ScopeLoggers {
	return s.Register(LOG_DELETE, callback)
}

func (s *ScopeLoggers) Query(callback func(action string, scope *Scope)) *ScopeLoggers {
	return s.Register(LOG_QUERY, callback)
}

func (s *ScopeLoggers) Exec(callback func(action string, scope *Scope)) *ScopeLoggers {
	return s.Register(LOG_EXEC, callback)
}

func (s *ScopeLoggers) Many(callback func(action string, scope *Scope), actions ...string) *ScopeLoggers {
	for _, action := range actions {
		s.Register(action, callback)
	}
	return s
}

func (s *ScopeLoggers) Crud(callback func(action string, scope *Scope)) *ScopeLoggers {
	return s.Many(callback, LOG_CREATE, LOG_UPDATE, LOG_DELETE)
}

func (s *ScopeLoggers) All(callback func(action string, scope *Scope)) *ScopeLoggers {
	return s.Many(callback, LOG_CREATE, LOG_UPDATE, LOG_DELETE, LOG_QUERY, LOG_EXEC)
}

func (s *ScopeLoggers) Call(action string, scope *Scope) {
	if s.callbacks == nil {
		return
	}
	if _, ok := s.callbacks[action]; !ok {
		return
	}

	if s.Async {
		go func() {
			for _, cb := range s.callbacks[action] {
				cb(action, scope)
			}
		}()
	} else {
		for _, cb := range s.callbacks[action] {
			cb(action, scope)
		}
	}
}

func scopeLoggerKey(realTableName string) string {
	return "aorm:logger:" + realTableName
}
