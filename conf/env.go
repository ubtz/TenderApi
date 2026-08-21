package config

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

func LoadEnvFile(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" || os.Getenv(name) != "" {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if err := os.Setenv(name, value); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if environment := strings.TrimSpace(os.Getenv("APP_ENV")); environment != "" {
		Env = environment
	}
	return nil
}
