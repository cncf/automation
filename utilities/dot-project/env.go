package projects

import (
	"bufio"
	"os"
	"strings"
)

// LoadDotEnv loads KEY=VALUE pairs from the env file at path into the process
// environment, returning the keys that were applied. It never overrides
// variables that are already set, so real environment values always take
// precedence. A missing file is not an error (returns nil, nil). Surrounding
// single or double quotes are stripped and a leading "export " is ignored;
// blank lines and lines beginning with "#" are skipped. Inline comments are
// not stripped, so a value like "ghp_xxx # note" is kept verbatim.
func LoadDotEnv(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var applied []string
	scanner := bufio.NewScanner(f)
	// Allow generously long lines (tokens can be long).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue // no key, or malformed
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if key == "" {
			continue
		}
		// Strip a single matching pair of surrounding quotes.
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		// Never override an already-set environment variable.
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			return applied, err
		}
		applied = append(applied, key)
	}
	if err := scanner.Err(); err != nil {
		return applied, err
	}
	return applied, nil
}
