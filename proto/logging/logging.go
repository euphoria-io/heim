package logging

import (
	"io"
	"log"
	"os"

	"euphoria.io/scope"
)

type logCtxKey int

const logCtx logCtxKey = 0
const logFlags = log.LstdFlags

func Logger(ctx scope.Context) *log.Logger {
	if logger, ok := ctx.Get(logCtx).(*log.Logger); ok {
		return logger
	}
	return log.New(os.Stdout, "[???] ", logFlags)
}

func LoggingContext(ctx scope.Context, w io.Writer, prefix string) scope.Context {
	logger := log.New(w, prefix, logFlags)
	ctx.Set(logCtx, logger)
	return ctx
}
