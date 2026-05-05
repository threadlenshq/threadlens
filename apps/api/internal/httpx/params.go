package httpx

import "strconv"

func ParsePositiveInt(value string, fallback int, max int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if max > 0 && parsed > max {
		return max
	}
	return parsed
}
