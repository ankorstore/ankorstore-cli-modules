package errorhandling

import (
	"os"

	"github.com/go-errors/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Check(err error, prefix string, level string) {
	if err == nil {
		return
	}

	var entry *zerolog.Event

	switch level {
	case "fatal":
		entry = log.Fatal().Err(err)
	case "error":
		entry = log.Error().Err(err)
	case "warn":
		entry = log.Warn().Err(err)
	}
	var es *errors.Error
	if errors.As(err, &es) {
		entry.Str("stacktrace", es.ErrorStack())
	}
	if len(prefix) > 0 {
		entry.Msg(prefix)
	} else {
		entry.Send()
	}

	if level == "fatal" {
		os.Exit(1)
	}
}

func CheckFatal(err error, prefixes ...string) {
	Check(err, handlePrefix(prefixes...), "fatal")
}

func CheckError(err error, prefixes ...string) {
	Check(err, handlePrefix(prefixes...), "error")
}

func CheckWarn(err error, prefixes ...string) {
	Check(err, handlePrefix(prefixes...), "warn")
}

func handlePrefix(prefixes ...string) string {
	var prefix string
	prefix = ""
	for _, p := range prefixes {
		prefix += p
	}
	return prefix
}
