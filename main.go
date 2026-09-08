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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.RulesEnabled {
		// Сработавшие правила передаются из подписчика (вызывается синхронно из цикла
		// опроса) единственному воркеру через буферизованный канал, чтобы медленное
		// или зависшее выполнение действий не блокировало опрос дома.
		fired := make(chan []rules.Rule, 16)
		pl.Subscribe(rulesSubscriber(engine, ruleStore, fired, logger))
		go runRulesWorker(ctx, fired, runner, events, pl, logger)
	} else {
		logger.Info("автоматизации выключены (RULES_ENABLED=false)")
	}
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
	shutdownDone := make(chan struct{})
	go func() {
		<-ctx.Done()
		_ = httpServer.Shutdown(context.Background())
		close(shutdownDone)
	}()
	serveErr := httpServer.ListenAndServe()
	<-shutdownDone // дожидаемся, пока Shutdown реально завершит все запросы в работе
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return serveErr
	}
	return nil
}

// rulesSubscriber возвращает обработчик снимков для pl.Subscribe: считает
// сработавшие правила синхронно (каждый снимок оценивается движком), затем
// передаёт пачку в очередь на выполнение воркеру. Паника при оценке правил
// логируется и не убивает процесс. Если очередь переполнена (воркер отстаёт),
// пачка отбрасывается с предупреждением — следующий снимок оценит правила заново.
func rulesSubscriber(engine *rules.Engine, store *rules.Store, fired chan<- []rules.Rule, logger *slog.Logger) func(snapshot.Snapshot) {
	return func(s snapshot.Snapshot) {
		defer func() {
			if v := recover(); v != nil {
				logger.Error("паника в цикле правил", "recover", v)
			}
		}()
		batch := engine.Tick(store.List(), s)
		if len(batch) == 0 {
			return
		}
		select {
		case fired <- batch:
		default:
			logger.Warn("очередь выполнения правил переполнена, пачка отброшена", "правил", len(batch))
		}
	}
}

// runRulesWorker — единственный воркер, исполняющий пачки сработавших правил по
// очереди, пока не отменят ctx.
func runRulesWorker(ctx context.Context, fired <-chan []rules.Rule, runner *rules.Runner, events *rules.EventLog, pl *poller.Poller, logger *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case batch := <-fired:
			for _, r := range batch {
				executeFiredRule(r, runner, events, logger)
			}
			pl.Kick()
		}
	}
}

// executeFiredRule выполняет действия одного сработавшего правила со своим
// таймаутом и восстановлением после паники, пишет событие в журнал.
func executeFiredRule(r rules.Rule, runner *rules.Runner, events *rules.EventLog, logger *slog.Logger) {
	defer func() {
		if v := recover(); v != nil {
			logger.Error("паника при выполнении правила", "rule", r.Name, "recover", v)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	details, err := runner.Execute(ctx, r)
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
}
