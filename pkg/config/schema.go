package config

import (
	_ "embed"
	"encoding/json"

	"github.com/qri-io/jsonschema"
)

//go:embed schema/ankor-config.schema.json
var ankorConfigSchemaData []byte

func GetAnkorConfigSchema() (jsonschema.Schema, error) {
	schema := jsonschema.Schema{}
	err := json.Unmarshal(ankorConfigSchemaData, &schema)
	return schema, err
}
