package config

import (
	"context"
	"fmt"
	"github.com/ankorstore/ankorstore-cli-modules/pkg/filesystem"
	"github.com/ankorstore/ankorstore-cli-modules/pkg/util"
	"github.com/go-errors/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

var (
	reset   = "\033[0m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[45m"
)

func InitConfig(flags *pflag.FlagSet) error {
	cfgFile, err := flags.GetString("config")
	if err != nil {
		return errors.Wrap(err, 0)
	}
	dirs := util.NewDirs()
	confDir := dirs.GetConfigDir()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(confDir)
		viper.SetConfigName(util.AppName)
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func validateConfigAgainstSchema(parsedConfig map[string]interface{}) error {
	schema, err := GetAnkorConfigSchema()
	if err != nil {
		return errors.Errorf("Failed to parse config schema: %s", err)
	}

	result := schema.Validate(context.Background(), parsedConfig)

	if !result.IsValid() {
		var errorMessages string
		for _, errDesc := range *result.Errs {
			errorMessages += fmt.Sprintf("\n - %s: %s", errDesc.PropertyPath, errDesc.Message)
		}
		return errors.Errorf("Parsed configuration is not valid %s", errorMessages)
	}
	return nil
}

func InitLogger(quiet bool, noColor bool) {
	dirs := util.NewDirs()
	logDir := dirs.GetLogsDir()

	filesystem.CreateFolder(logDir)

	logPath := filepath.Join(logDir, util.AppName+".log")
	fileWriter, _ := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if quiet {
		log.Logger = zerolog.New(fileWriter).With().Timestamp().Logger()
	} else {
		noColor := noColor || viper.GetBool("logging.noColor")

		consoleWriter := zerolog.ConsoleWriter{
			Out:          os.Stderr,
			NoColor:      noColor,
			PartsExclude: []string{"time"},
		}

		consoleWriter.FormatLevel = func(i interface{}) string {
			if i == nil {
				return ""
			}
			m := fmt.Sprintf("%s", "")
			switch i {
			case zerolog.LevelTraceValue:
				m = colorize(m, magenta, false, consoleWriter.NoColor)
			case zerolog.LevelDebugValue:
				m = colorize(m, blue, false, consoleWriter.NoColor)
			case zerolog.LevelInfoValue:
				m = colorize(m, green, false, consoleWriter.NoColor)
			case zerolog.LevelWarnValue:
				m = colorize(m, yellow, false, consoleWriter.NoColor)
			case zerolog.LevelErrorValue:
				m = colorize(m, red, true, consoleWriter.NoColor)
			case zerolog.LevelFatalValue:
				m = colorize(m, red, true, consoleWriter.NoColor)
			case zerolog.LevelPanicValue:
				m = colorize(m, red, true, consoleWriter.NoColor)
			}
			return m
		}

		consoleWriter.FormatMessage = func(i interface{}) string {
			m := fmt.Sprintf("%s%s\x1b[0m", i, reset)
			return m
		}

		multi := zerolog.MultiLevelWriter(consoleWriter, fileWriter)

		log.Logger = zerolog.New(multi).With().Timestamp().Logger()
	}

	level := viper.GetString("logging.level")

	switch strings.ToLower(level) {
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}
}

func colorize(s interface{}, c string, bold, disabled bool) string {
	if disabled {
		return fmt.Sprintf("%s", s)
	}
	var b string
	if bold {
		b = fmt.Sprintf("\u001B[%dm", 1)
	}
	return fmt.Sprintf("%s%s%v", b, c, strings.ToUpper(fmt.Sprintf("%s", s)))
}
