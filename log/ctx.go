package log

import "context"

func For(r interface{ Context() context.Context }) Logger {
	if l, ok := r.Context().Value(ContextKey).(Logger); ok {
		return l
	}
	return New() // return root logger if none is found
}

const ContextKey loggerKey = "logger"

type loggerKey string
