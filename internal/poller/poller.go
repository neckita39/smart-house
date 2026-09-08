// Package poller периодически запрашивает user/info и раздаёт снимки состояния дома.
package poller

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"smarthome/internal/snapshot"
)

type Source interface {
	UserInfo(ctx context.Context) (json.RawMessage, error)
}

type Poller struct {
	src      Source
	interval time.Duration
	log      *slog.Logger

	mu     sync.RWMutex
	latest snapshot.Snapshot
	has    bool
	subs   []func(snapshot.Snapshot)
	kick   chan struct{}
}

func New(src Source, interval time.Duration, log *slog.Logger) *Poller {
	if log == nil {
		log = slog.Default()
	}
	return &Poller{src: src, interval: interval, log: log, kick: make(chan struct{}, 1)}
}

// Subscribe регистрирует обработчик, вызываемый после каждого успешного опроса.
func (p *Poller) Subscribe(fn func(snapshot.Snapshot)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subs = append(p.subs, fn)
}

func (p *Poller) Latest() (snapshot.Snapshot, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.latest, p.has
}

// Refresh выполняет опрос немедленно и возвращает новый снимок.
func (p *Poller) Refresh(ctx context.Context) (snapshot.Snapshot, error) {
	raw, err := p.src.UserInfo(ctx)
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	s, err := snapshot.Parse(raw, time.Now())
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	p.mu.Lock()
	p.latest, p.has = s, true
	subs := append([]func(snapshot.Snapshot){}, p.subs...)
	p.mu.Unlock()
	for _, fn := range subs {
		fn(s)
	}
	return s, nil
}

// Kick просит цикл опросить дом вне расписания (после действий пользователя).
func (p *Poller) Kick() {
	select {
	case p.kick <- struct{}{}:
	default:
	}
}

// Run опрашивает дом раз в interval до отмены контекста; ошибки логируются,
// последний удачный снимок сохраняется.
func (p *Poller) Run(ctx context.Context) {
	t := time.NewTicker(p.interval)
	defer t.Stop()
	for {
		if _, err := p.Refresh(ctx); err != nil && ctx.Err() == nil {
			p.log.Warn("опрос дома не удался", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-p.kick:
		}
	}
}
