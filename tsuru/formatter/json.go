package formatter

import (
	"encoding/json"
	"io"
)

func JSON(writer io.Writer, data interface{}) error {
	enc := json.NewEncoder(writer)
	enc.SetIndent("  ", "  ")
	return enc.Encode(data)
}
