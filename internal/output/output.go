package output

import (
	"encoding/json"
	"fmt"
	"io"
)

func Write(out io.Writer, value any, jsonOutput bool, text func() string) error {
	if jsonOutput {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}

	_, err := fmt.Fprintln(out, text())
	return err
}
