// Package config читает настройки приложения из .env и переменных окружения.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

type Config struct {
	ClientID     string
	ClientSecret string
	Port         string
	DataDir      string
	Host         string
	AllowedHosts []string
	PollSeconds  int
	RulesEnabled bool
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
	pollSeconds, err := strconv.Atoi(get("POLL_SECONDS", "10"))
	if err != nil {
		return Config{}, fmt.Errorf("POLL_SECONDS: ожидается целое число, получено %q", get("POLL_SECONDS", "10"))
	}
	if pollSeconds < 3 {
		pollSeconds = 3
	}
	cfg := Config{
		ClientID:     get("YANDEX_CLIENT_ID", ""),
		ClientSecret: get("YANDEX_CLIENT_SECRET", ""),
		Port:         get("PORT", "8080"),
		DataDir:      get("DATA_DIR", "data"),
		Host:         get("HOST", "127.0.0.1"),
		AllowedHosts: parseAllowedHosts(get("ALLOWED_HOSTS", "")),
		PollSeconds:  pollSeconds,
		RulesEnabled: !slices.Contains([]string{"false", "0", "no", "off"}, strings.ToLower(get("RULES_ENABLED", "true"))),
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return Config{}, errors.New("нужны YANDEX_CLIENT_ID и YANDEX_CLIENT_SECRET (в .env или переменных окружения)")
	}
	return cfg, nil
}

// parseAllowedHosts разбирает список хостов через запятую: обрезает пробелы,
// отбрасывает пустые элементы.
func parseAllowedHosts(raw string) []string {
	if raw == "" {
		return nil
	}
	var hosts []string
	for _, h := range strings.Split(raw, ",") {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		hosts = append(hosts, h)
	}
	return hosts
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
