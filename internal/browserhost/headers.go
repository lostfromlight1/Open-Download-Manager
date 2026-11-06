package browserhost

import (
	"fmt"
	"strings"
	"unicode"
)

// MaxAuthHeaderBytes caps Cookie / Referer strings from the extension.
const MaxAuthHeaderBytes = 8 * 1024

// ValidateAuthHeader checks an optional Cookie or Referer value from NM.
// Empty is allowed. Values must not contain CR/LF or other control characters.
func ValidateAuthHeader(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if len(value) > MaxAuthHeaderBytes {
		return fmt.Errorf("%s too large (%d bytes, max %d)", name, len(value), MaxAuthHeaderBytes)
	}
	for _, r := range value {
		if r == '\r' || r == '\n' || (unicode.IsControl(r) && r != '\t') {
			return fmt.Errorf("%s contains invalid control characters", name)
		}
	}
	return nil
}
