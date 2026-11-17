package formatter

import (
	"encoding/json"
	"fmt"
	"io"
)

func JSON(writer io.Writer, data any) error {
	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// EmptyJSON writes an empty JSON array, used when a --json listing has no items.
func EmptyJSON(writer io.Writer) error {
	_, err := fmt.Fprintln(writer, "[]")
	return err
}
