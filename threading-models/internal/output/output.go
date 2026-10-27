package output

import (
	"encoding/json"
	"io"

	"threading-models/internal/experiment"
)

// Write emits exactly one JSON line after the measurement and verification.
func Write(w io.Writer, record experiment.Record) error {
	return json.NewEncoder(w).Encode(record)
}
