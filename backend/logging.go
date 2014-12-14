package backend

import (
	"log"
	"os"

	"golang.org/x/net/context"
)

type logCtxKey int

const logCtx logCtxKey = 0
const logFlags = log.LstdFlags

func Logger(ctx context.Context) *log.Logger {
	if logger, ok := ctx.Value(logCtx).(*log.Logger); ok {
		return logger
	}
	return log.New(os.Stdout, "[???] ", logFlags)
}

func LoggingContext(parent context.Context, prefix string) context.Context {
	logger := log.New(os.Stdout, prefix, logFlags)
	return context.WithValue(parent, logCtx, logger)
}
