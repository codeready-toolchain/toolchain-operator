package toolchain

import (
	"fmt"
	"time"
)

// GenerateName return the given name with a suffix based on the current time (UnixNano)
func GenerateName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
