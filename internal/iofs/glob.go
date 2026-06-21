package iofs

import (
	"path/filepath"
	"regexp"
	"strings"
)

func hasDoubleStar(pattern string) bool {
	return strings.Contains(pattern, "**")
}

// globBaseDir returns the longest path prefix of pattern that contains no wildcard
// characters, so callers can scope a directory walk instead of scanning from root.
func globBaseDir(pattern string) string {
	idx := strings.IndexAny(pattern, "*?")
	if idx == -1 {
		return pattern
	}
	return filepath.Dir(pattern[:idx])
}

// GlobRegexp compiles pattern (glob syntax plus a doublestar "**" segment matching zero
// or more path segments) into a regexp matching whole paths. Exported so other FS
// implementations (e.g. iofstest) can match the same semantics as New()'s Glob.
func GlobRegexp(pattern string) *regexp.Regexp {
	var sb strings.Builder
	sb.WriteString("^")
	for i := 0; i < len(pattern); {
		switch {
		case strings.HasPrefix(pattern[i:], "**/"):
			sb.WriteString("(.*/)?")
			i += 3
		case strings.HasPrefix(pattern[i:], "**"):
			sb.WriteString(".*")
			i += 2
		case pattern[i] == '*':
			sb.WriteString("[^/]*")
			i++
		case pattern[i] == '?':
			sb.WriteString("[^/]")
			i++
		default:
			sb.WriteString(regexp.QuoteMeta(string(pattern[i])))
			i++
		}
	}
	sb.WriteString("$")
	return regexp.MustCompile(sb.String())
}
