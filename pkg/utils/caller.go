package utils

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aarioai/aago/pkg/types"
)

func Caller(skip int) string {
	// Skip Caller itself. When walking further up, add only 1 so another skip does not pass over the user's frame.
	skip++
	for {
		_, file, line, ok := runtime.Caller(skip)
		if !ok {
			return ""
		}
		slash := filepath.ToSlash(file)
		// module cache: /aago@v1.0.0/ ; source tree: /aago/
		if strings.Contains(slash, "/aago/") || strings.Contains(slash, "/aago@") {
			skip++
			continue
		}
		parts := strings.Split(slash, "/")
		if len(parts) > 2 {
			file = parts[len(parts)-2] + "/" + parts[len(parts)-1]
		} else {
			file = slash
		}
		return file + ":" + types.FormatInt(line)
	}
}
