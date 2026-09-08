// Веб-пульт умного дома Яндекса: локальный сервер, раздающий UI и проксирующий API Яндекса.
package main

import (
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"smarthome/internal/auth"
	"smarthome/internal/config"
	"smarthome/internal/macros"
	"smarthome/internal/server"
	"smarthome/internal/yandex"
	"smarthome/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("завершение с ошибкой", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load(".env")
	if err != nil {
		return err
	}

	oauth := &yandex.OAuth{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret}
	tokens, err := auth.NewManager(oauth, &auth.Store{Path: filepath.Join(cfg.DataDir, "token.json")})
	if err != nil {
		return err
	}
	macroStore, err := macros.NewStore(filepath.Join(cfg.DataDir, "macros.json"))
	if err != nil {
		return err
	}

	ui, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		return err
	}
	if _, err := fs.Stat(ui, "index.html"); err != nil {
		logger.Warn("фронтенд не собран (нет web/dist/index.html) — выполните `make web` и пересоберите")
		ui = nil
	}

	srv := &server.Server{
		Auth:   tokens,
		Home:   &yandex.Client{Tokens: tokens, Log: logger},
		Macros: macroStore,
		UI:     ui,
		Log:    logger,
	}

	addr := "127.0.0.1:" + cfg.Port
	logger.Info("сервер запущен", "url", "http://localhost:"+cfg.Port, "authorized", tokens.Authorized())
	return http.ListenAndServe(addr, srv.Handler())
}
