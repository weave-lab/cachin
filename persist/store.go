package persist

import (
	"encoding/base64"
	"strings"
	"time"
)

// SafeKey takes an arbitrary string and converts it to a string that contains only the following characters
// 0-9, a-z, A-Z, -, _, .
func SafeKey(key string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(key))
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "_")
	encoded = strings.ReplaceAll(encoded, "=", ".")
	return encoded
}

// rawData wraps raw bytes in a struct along with the last update time. This can be used to make storing data in an
// external data store easier
type rawData struct {
	LastSet time.Time
	Raw     []byte
}
