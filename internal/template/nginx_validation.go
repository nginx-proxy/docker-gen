package template

import (
	"fmt"
	"strings"
)

type hstsDirectives struct {
	MaxAge            int
	IncludeSubDomains bool
	Preload           bool
}

// mustParseHSTS validates and normalizes the Strict-Transport-Security header value according to its specification.
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Strict-Transport-Security
func mustParseHSTS(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("HSTS value cannot be empty")
	}

	if value == "off" {
		return value, nil
	}

	var directives hstsDirectives

	for directive := range strings.SplitSeq(value, ";") {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, "max-age=") {
			_, err := fmt.Sscanf(directive, "max-age=%d", &directives.MaxAge)
			if err != nil {
				return "", fmt.Errorf("invalid max-age directive: %s", directive)
			}
		} else if directive == "includeSubDomains" {
			directives.IncludeSubDomains = true
		} else if directive == "preload" {
			directives.Preload = true
		} else {
			return "", fmt.Errorf("unknown HSTS directive: %s", directive)
		}
	}

	if directives.MaxAge <= 0 {
		return "", fmt.Errorf("max-age directive must be greater than 0")
	}

	if directives.Preload {
		if !directives.IncludeSubDomains {
			return "", fmt.Errorf("preload directive requires includeSubDomains to be set")
		}
		if directives.MaxAge < 31536000 {
			return "", fmt.Errorf("preload directive requires max-age to be at least 31536000 seconds (1 year)")
		}
		return fmt.Sprintf("max-age=%d; includeSubDomains; preload", directives.MaxAge), nil
	}

	if directives.IncludeSubDomains {
		return fmt.Sprintf("max-age=%d; includeSubDomains", directives.MaxAge), nil
	}

	return fmt.Sprintf("max-age=%d", directives.MaxAge), nil
}
