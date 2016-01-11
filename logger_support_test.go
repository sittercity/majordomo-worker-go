package majordomo_worker

import (
	"fmt"
)

type testLogger struct {
	debugs []map[string]interface{}
	errors []map[string]interface{}
}

// A lot of this is stolen from https://github.com/go-kit/kit/blob/master/log/json_logger.go
func (l *testLogger) Log(keyvals ...interface{}) error {
	n := (len(keyvals) + 1) / 2 // +1 to handle case when len is odd
	m := make(map[string]interface{}, n)
	for i := 0; i < len(keyvals); i += 2 {
		k := keyvals[i]
		var v interface{} = ErrMissingValue
		if i+1 < len(keyvals) {
			v = keyvals[i+1]
		}
		l.merge(m, k, v)
	}

	level := m["level"]
	delete(m, "level")
	if level == "debug" {
		l.debugs = append(l.debugs, m)
	} else if level == "error" {
		l.errors = append(l.errors, m)
	}

	return nil
}

func (l *testLogger) merge(dst map[string]interface{}, k, v interface{}) {
	var key string
	switch x := k.(type) {
	case string:
		key = x
	default:
		key = fmt.Sprint(x)
	}

	var value string
	switch x := v.(type) {
	case string:
		value = x
	default:
		value = fmt.Sprint(x)
	}

	dst[key] = value
}

func (l *testLogger) reset() {
	l.debugs = make([]map[string]interface{}, 0)
	l.errors = make([]map[string]interface{}, 0)
}
