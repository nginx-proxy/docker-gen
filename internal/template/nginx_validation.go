package template

import (
	"fmt"
	"slices"
	"strings"
)

type hstsDirectives struct {
	MaxAge            int
	IncludeSubDomains bool
	Preload           bool
}

// nginxMustParseHSTS validates and normalizes a Strict-Transport-Security header value.
func nginxMustParseHSTS(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)

	if value == "" {
		return "", fmt.Errorf("HSTS value cannot be empty")
	}

	var directives hstsDirectives

	for directive := range strings.SplitSeq(value, ";") {
		directive = strings.TrimSpace(directive)
		if maxAgeStr, ok := strings.CutPrefix(directive, "max-age="); ok {
			if maxAgeStr == "" {
				return "", fmt.Errorf("invalid Strict-Transport-Security max-age directive: %s", directive)
			}
			for _, r := range maxAgeStr {
				if r < '0' || r > '9' {
					return "", fmt.Errorf("invalid Strict-Transport-Security max-age directive: %s", directive)
				}
			}
			_, err := fmt.Sscanf(maxAgeStr, "%d", &directives.MaxAge)
			if err != nil {
				return "", fmt.Errorf("invalid Strict-Transport-Security max-age directive: %s", directive)
			}
		} else if directive == "includesubdomains" {
			directives.IncludeSubDomains = true
		} else if directive == "preload" {
			directives.Preload = true
		} else {
			return "", fmt.Errorf("unknown Strict-Transport-Security directive: %s", directive)
		}
	}

	if directives.MaxAge <= 0 {
		return "", fmt.Errorf("Strict-Transport-Security max-age directive must be greater than 0")
	}

	if directives.Preload {
		if !directives.IncludeSubDomains {
			return "", fmt.Errorf("Strict-Transport-Security preload directive requires includeSubDomains to be set")
		}
		if directives.MaxAge < 31536000 {
			return "", fmt.Errorf("Strict-Transport-Security preload directive requires max-age to be at least 31536000 seconds (1 year)")
		}
		return fmt.Sprintf("max-age=%d; includeSubDomains; preload", directives.MaxAge), nil
	}

	if directives.IncludeSubDomains {
		return fmt.Sprintf("max-age=%d; includeSubDomains", directives.MaxAge), nil
	}

	return fmt.Sprintf("max-age=%d", directives.MaxAge), nil
}

// nginxMustParseLoadbalance validates and normalizes the Nginx directives for the various supported loadbalancing methods.
// https://docs.nginx.com/nginx/admin-guide/load-balancer/http-load-balancer/#method
func nginxMustParseLoadbalance(value string) (string, error) {
	value = strings.TrimSuffix(value, ";")
	methodAndParameters := strings.Split(value, " ")
	method := strings.TrimSpace(methodAndParameters[0])

	switch method {
	case "hash":
		return parseHashLoadBalanceMethod(methodAndParameters)
	case "ip_hash", "least_conn":
		return parseUnparameterizedLoadbalanceMethod(methodAndParameters)
	case "least_time":
		return parseLeastTimeLoadbalanceMethod(methodAndParameters)
	case "random":
		return parseRandomLoadbalanceMethod(methodAndParameters)
	default:
		validMethods := []string{"hash", "ip_hash", "least_conn", "least_time", "random"}
		return "", fmt.Errorf("invalid loadbalance directive: %s. Valid values are: %v", method, validMethods)
	}
}

// parseHashLoadBalanceMethod validates and normalizes the Nginx hash load balancing method.
func parseHashLoadBalanceMethod(value []string) (string, error) {
	method := "hash"

	var firstParameter, secondParameter string
	var err error

	switch len(value) {
	case 3:
		secondParameter = strings.TrimSpace(value[2])
		if secondParameter != "consistent" {
			return "", fmt.Errorf("%s loadbalance method does not take any second parameter other than 'consistent'", method)
		}
		fallthrough
	case 2:
		if firstParameter, err = nginxQuote(value[1]); err != nil {
			return "", err
		}
		fallthrough
	case 1:
		// no parameters
	default:
		return "", fmt.Errorf("%s loadbalance method does not take more than two parameters", method)
	}

	if secondParameter != "" && firstParameter != "" {
		return fmt.Sprintf("%s %s %s", method, firstParameter, secondParameter), nil
	}
	if firstParameter != "" {
		return fmt.Sprintf("%s %s", method, firstParameter), nil
	}
	return fmt.Sprintf("%s", method), nil
}

// parseUnparameterizedLoadbalanceMethod validates and normalizes the Nginx load balancing methods that do not take any parameters (ip_hash and least_conn).
func parseUnparameterizedLoadbalanceMethod(value []string) (string, error) {
	method := strings.TrimSpace(value[0])
	if len(value) != 1 {
		return "", fmt.Errorf("%s loadbalance method does not take any parameters", method)
	}
	return method, nil
}

// parseLeastTimeLoadbalanceMethod validates and normalizes the Nginx least_time load balancing method.
func parseLeastTimeLoadbalanceMethod(value []string) (string, error) {
	method := "least_time"

	if len(value) < 2 {
		return "", fmt.Errorf("%s loadbalance directive requires a parameter", method)
	}

	parameter := strings.Join(value[1:], " ")
	parameter = strings.TrimSpace(parameter)
	validParameters := []string{"header", "last_byte", "last_byte inflight"}
	if slices.Contains(validParameters, parameter) {
		return fmt.Sprintf("%s %s", method, parameter), nil
	}

	return "", fmt.Errorf("invalid least_time loadbalance method parameter: %s. Valid values are: %v", parameter, validParameters)
}

// parseRandomLoadbalanceMethod validates and normalizes the Nginx random load balancing method.
func parseRandomLoadbalanceMethod(value []string) (string, error) {
	method := "random"

	var firstParameter, secondParameter string

	switch len(value) {
	case 3:
		validSecondParameters := []string{"least_conn", "least_time=header", "least_time=last_byte"}
		secondParameter = strings.TrimSpace(value[2])
		if !slices.Contains(validSecondParameters, secondParameter) {
			return "", fmt.Errorf("%s loadbalance method with 'two' parameter only accepts %v as the second parameter", method, validSecondParameters)
		}
		fallthrough
	case 2:
		firstParameter = strings.TrimSpace(value[1])
		if firstParameter != "two" {
			return "", fmt.Errorf("%s loadbalance method does not take any first parameter other than 'two'", method)
		}
		fallthrough
	case 1:
		// no parameters
	default:
		return "", fmt.Errorf("%s loadbalance method does not take more than two parameters", method)
	}

	if firstParameter != "" && secondParameter != "" {
		return fmt.Sprintf("%s %s %s", method, firstParameter, secondParameter), nil
	}
	if firstParameter != "" {
		return fmt.Sprintf("%s %s", method, firstParameter), nil
	}
	return fmt.Sprintf("%s", method), nil
}
