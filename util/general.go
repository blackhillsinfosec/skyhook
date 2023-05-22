package util

import (
	"gopkg.in/yaml.v3"
	"os"
)

// UnmarshalFileInto handles YAML config file processing.
func UnmarshalFileInto(file *string, dest interface{}) (err error) {
	b, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, dest)
}
