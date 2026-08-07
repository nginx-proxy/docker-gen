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

// nginxMustParseHSTS validates and normalizes a Strict-Transport-Security header value.
// It also accepts the special value "off" (nginx-proxy convention) to indicate the header should not be set.
func nginxMustParseHSTS(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)

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
			maxAgeStr := strings.TrimPrefix(directive, "max-age=")
			if maxAgeStr == "" {
				return "", fmt.Errorf("invalid max-age directive: %s", directive)
			}
			for _, r := range maxAgeStr {
				if r < '0' || r > '9' {
					return "", fmt.Errorf("invalid max-age directive: %s", directive)
				}
			}
			_, err := fmt.Sscanf(maxAgeStr, "%d", &directives.MaxAge)
			if err != nil {
				return "", fmt.Errorf("invalid max-age directive: %s", directive)
			}
		} else if directive == "includesubdomains" {
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
