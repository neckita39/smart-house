// Package config читает настройки приложения из .env и переменных окружения.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ClientID     string
	ClientSecret string
	Port         string
	DataDir      string
}

// Load читает файл envPath (если он есть) и переменные окружения;
// окружение имеет приоритет над файлом.
func Load(envPath string) (Config, error) {
	fileVals, err := readDotEnv(envPath)
	if err != nil {
		return Config{}, err
	}
	get := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		if v := fileVals[key]; v != "" {
			return v
		}
		return def
	}
	cfg := Config{
		ClientID:     get("YANDEX_CLIENT_ID", ""),
		ClientSecret: get("YANDEX_CLIENT_SECRET", ""),
		Port:         get("PORT", "8080"),
		DataDir:      get("DATA_DIR", "data"),
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return Config{}, errors.New("нужны YANDEX_CLIENT_ID и YANDEX_CLIENT_SECRET (в .env или переменных окружения)")
	}
	return cfg, nil
}

func readDotEnv(path string) (map[string]string, error) {
	vals := map[string]string{}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return vals, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: ожидается строка вида KEY=VALUE", path, n)
		}
		vals[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(val), `"'`)
	}
	return vals, sc.Err()
}
