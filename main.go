// Веб-пульт умного дома Яндекса: локальный сервер, раздающий UI и проксирующий API Яндекса.
package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"smarthome/internal/auth"
	"smarthome/internal/config"
	"smarthome/internal/macros"
	"smarthome/internal/poller"
	"smarthome/internal/rules"
	"smarthome/internal/server"
	"smarthome/internal/snapshot"
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
	client := &yandex.Client{Tokens: tokens, Log: logger}

	ruleStore, err := rules.NewStore(filepath.Join(cfg.DataDir, "rules.json"))
	if err != nil {
		return err
	}
	events, err := rules.NewEventLog(filepath.Join(cfg.DataDir, "events.jsonl"), 500)
	if err != nil {
		return err
	}
	engine := rules.NewEngine(nil)
	runner := &rules.Runner{Home: client, Macros: macroStore}
	pl := poller.New(client, time.Duration(cfg.PollSeconds)*time.Second, logger)
	if cfg.RulesEnabled {
		pl.Subscribe(func(s snapshot.Snapshot) {
			for _, r := range engine.Tick(ruleStore.List(), s) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				details, err := runner.Execute(ctx, r)
				cancel()
				ev := rules.Event{Time: time.Now(), RuleID: r.ID, RuleName: r.Name, OK: err == nil, Details: details}
				if err != nil {
					ev.Details += " — " + err.Error()
					logger.Warn("правило выполнено с ошибкой", "rule", r.Name, "err", err, "details", details)
				} else {
					logger.Info("правило сработало", "rule", r.Name, "details", details)
				}
				if err := events.Append(ev); err != nil {
					logger.Warn("не удалось записать событие", "err", err)
				}
				pl.Kick()
			}
		})
	} else {
		logger.Info("автоматизации выключены (RULES_ENABLED=false)")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go pl.Run(ctx)

	ui, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		return err
	}
	if _, err := fs.Stat(ui, "index.html"); err != nil {
		logger.Warn("фронтенд не собран (нет web/dist/index.html) — выполните `make web` и пересоберите")
		ui = nil
	}

	apiServer := &server.Server{
		Auth:         tokens,
		Home:         client,
		Macros:       macroStore,
		Snapshots:    pl,
		Rules:        ruleStore,
		Runner:       runner,
		Events:       events,
		Engine:       engine,
		UI:           ui,
		Log:          logger,
		AllowedHosts: cfg.AllowedHosts,
	}

	addr := cfg.Host + ":" + cfg.Port
	displayHost := cfg.Host
	if displayHost == "0.0.0.0" {
		displayHost = "127.0.0.1"
		logger.Info("сервер запущен", "url", "http://"+displayHost+":"+cfg.Port,
			"listen", "0.0.0.0:"+cfg.Port, "authorized", tokens.Authorized())
	} else {
		logger.Info("сервер запущен", "url", "http://"+displayHost+":"+cfg.Port,
			"authorized", tokens.Authorized())
	}

	httpServer := &http.Server{Addr: addr, Handler: apiServer.Handler()}
	go func() {
		<-ctx.Done()
		_ = httpServer.Shutdown(context.Background())
	}()
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
