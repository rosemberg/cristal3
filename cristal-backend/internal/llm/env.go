package llm

import "os"

// setenvIfEmpty sets an environment variable only if it's currently unset,
// so an operator-provided value in the environment always wins over config.
func setenvIfEmpty(key, value string) error {
	if os.Getenv(key) != "" {
		return nil
	}
	return os.Setenv(key, value)
}
