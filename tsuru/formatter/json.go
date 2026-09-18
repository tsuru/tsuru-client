package formatter

import (
	"encoding/json"
	"io"
)

func JSON(writer io.Writer, data any) error {
	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
