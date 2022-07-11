package util

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/rs/zerolog/log"
)

func CreateConfigFromTemplate(name string, data map[string]interface{}) error {
	return CreateConfigFromSourceTemplate(name, name, data)
}

func CreateConfigFromSourceTemplate(source, target string, data map[string]interface{}) error {
	path := filepath.Join(confDir, target)
	if strings.Contains(target, "/") {
		err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}
	s := fmt.Sprintf("templates/%s", source)
	return CreateFromSourceTemplate(s, path, TemplateFS, data)
}

func CreateFromSourceTemplate(source, target string, t embed.FS, data map[string]interface{}) error {
	if _, err := os.Stat(target); os.IsNotExist(err) || Force {
		log.Info().Msgf("Creating configuration file at %s", target)
		f, err := os.Create(target)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		t, err := template.ParseFS(t, source)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		err = t.Execute(f, data)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}
	return nil
}
