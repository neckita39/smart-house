# Автоматизации — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Локальные правила «если условия по датчикам/времени → действия», которые сервер проверяет на каждом опросе дома и выполняет сам; Claude создаёт правила через API, владелец включает/выключает и смотрит журнал во вкладке «Автоматизации».

**Architecture:** Опрос `user/info` переезжает с фронта на сервер (`internal/poller`, раз в 10 с) и превращается в снимок состояния (`internal/snapshot`). Движок (`internal/rules`) на каждом снимке вычисляет условия включённых правил, срабатывает по фронту с учётом `for_minutes` и кулдауна и выполняет действия через клиент Яндекса; события пишутся в `data/events.jsonl`. `/api/home` и `/api/catalog` отдают данные из снимка. Фронт получает вкладку «Автоматизации» в стиле «Плитки».

**Tech Stack:** Go 1.26 (стандартная библиотека), React 19 + Vite 7 (JS, plain CSS), существующие пакеты `internal/yandex`, `internal/macros`, `internal/catalog`, `internal/server`.

**Spec:** `docs/superpowers/specs/2026-09-08-automations-design.md` (основа — `docs/superpowers/specs/2026-09-08-smart-home-web-remote-design.md`).

## Global Constraints

- Go: только стандартная библиотека. Фронт: без новых npm-пакетов.
- Секреты не читать и не печатать (`.env`), не коммитить `.env`, `data/`, `build/`, `web/dist/*` (кроме `.gitkeep`), `.superpowers/`.
- Время — локальное время сервера (`time.Local`); окна `after > before` — через полночь.
- Условия правила объединяются по «И»; срабатывание по фронту (false→true) после `for_minutes`, повтор не раньше `cooldown_minutes` (по умолчанию 30) и только после сброса условий.
- Все действия над устройствами одного правила — одним запросом `devices/actions`.
- Сервер — единственный писатель `data/rules.json` и `data/events.jsonl`.
- Тексты для пользователя — по-русски. Иконки — только SVG из `web/src/icons.jsx`.
- Коммиты по-русски `тип: описание`, каждый заканчивается строками
  `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` и
  `Claude-Session: https://claude.ai/code/session_01UyvwkyeaJ8suoGaSG44T2X`.
- Перед коммитом: `gofmt -l .` пусто, `go vet ./...`, `go test ./...`; для фронта `cd web && npm run build` без предупреждений.
- На ноутбуке разработчика на 8765 работает живой сервер владельца — не убивать и не занимать порты 8765/8080; дымовые запуски на 18080 с `DATA_DIR=/tmp/smart-house-smoke`, свой процесс убивать.

## Структура файлов

```
internal/snapshot/snapshot.go        Snapshot, DeviceState, Parse(raw user/info)
internal/snapshot/snapshot_test.go
internal/poller/poller.go            Poller: цикл опроса, Latest(), Refresh(), подписчики
internal/poller/poller_test.go
internal/rules/rule.go               Rule, Condition, TimeWindow, Action, Validate(snapshot)
internal/rules/store.go              Store: CRUD в rules.json
internal/rules/engine.go             Evaluate (условия → результаты), Engine.Tick (фронт, for_minutes, кулдаун)
internal/rules/events.go             EventLog: events.jsonl + кольцо на 500
internal/rules/runner.go             Runner.Execute: действия → devices/actions / макрос / сценарий
internal/rules/*_test.go
internal/config/config.go            + PollSeconds (POLL_SECONDS), RulesEnabled (RULES_ENABLED)
internal/server/server.go            + поля Snapshots, Rules, Runner, Events; маршруты
internal/server/home_handlers.go     /api/home и /api/catalog из снимка; Refresh после действий
internal/server/rule_handlers.go     CRUD /api/rules, run, check, /api/events
internal/server/*_test.go
main.go                              запуск poller + цикл правил
web/src/Home.jsx                     третья вкладка «Автоматизации»
web/src/Automations.jsx              плитки правил, «Проверить», журнал
web/src/describeRule.js              человекочитаемое описание условий/действий
web/src/api.js                       rules, ruleCheck, ruleRun, events
web/src/styles.css                   стили вкладки
.claude/skills/smart-home/SKILL.md   раздел «Автоматизации»
README.md                            раздел «Автоматизации»
```

Зависимости: `snapshot` — ни от кого; `poller` → `snapshot`, `yandex`; `rules` → `snapshot`, `yandex`, `macros`; `server` → всё; `main` → всё.

## Образец `user/info` для тестов

Используется в тестах snapshot/poller/rules (константа `sampleUserInfo`):

```json
{"status":"ok","request_id":"r",
 "rooms":[{"id":"r-office","name":"Кабинет","devices":["sensor-office","blinds"]},
          {"id":"r-bedroom","name":"Спальня","devices":["fan","purifier"]}],
 "devices":[
  {"id":"sensor-office","name":"Датчик климата","type":"devices.types.sensor.climate","room":"r-office",
   "capabilities":[],
   "properties":[{"type":"devices.properties.float","parameters":{"instance":"temperature","unit":"unit.temperature.celsius"},"state":{"instance":"temperature","value":26.3}},
                 {"type":"devices.properties.float","parameters":{"instance":"humidity","unit":"unit.percent"},"state":{"instance":"humidity","value":48.9}}]},
  {"id":"blinds","name":"Жалюзи в кабинете","type":"devices.types.openable.curtain","room":"r-office",
   "capabilities":[{"type":"devices.capabilities.range","parameters":{"instance":"open","unit":"unit.percent","range":{"min":0,"max":100,"precision":1}},"state":{"instance":"open","value":100}},
                   {"type":"devices.capabilities.on_off","parameters":{},"state":{"instance":"on","value":false}}],
   "properties":[]},
  {"id":"fan","name":"Вентилятор","type":"devices.types.ventilation.fan","room":"r-bedroom",
   "capabilities":[{"type":"devices.capabilities.on_off","parameters":{},"state":{"instance":"on","value":false}},
                   {"type":"devices.capabilities.mode","parameters":{"instance":"fan_speed","modes":[{"value":"low"},{"value":"medium"},{"value":"high"}]},"state":{"instance":"fan_speed","value":"low"}}],
   "properties":[]},
  {"id":"purifier","name":"Очиститель воздуха","type":"devices.types.purifier","room":"r-bedroom",
   "capabilities":[{"type":"devices.capabilities.on_off","parameters":{},"state":{"instance":"on","value":true}}],
   "properties":[{"type":"devices.properties.float","parameters":{"instance":"pm2.5_density","unit":"unit.density.mcg_m3"},"state":{"instance":"pm2.5_density","value":3}}]},
  {"id":"tv","name":"Телевизор в гостиной","type":"devices.types.media_device.tv","room":"",
   "capabilities":[{"type":"devices.capabilities.on_off","parameters":{},"state":null}],
   "properties":[]}
 ],
 "scenarios":[{"id":"s1","name":"Я дома","is_active":true}]}
```

---

### Task 1: Снимок состояния дома

**Files:**
- Create: `internal/snapshot/snapshot.go`
- Test: `internal/snapshot/snapshot_test.go`

**Interfaces:**
- Produces:
  - `snapshot.DeviceState{ID, Name, Type, Room string; Props map[string]any; Caps map[string]any; CapTypes map[string]string; Modes map[string][]string; Ranges map[string]Range}`; `snapshot.Range{Min, Max float64; HasRange bool}`
  - `snapshot.Snapshot{At time.Time; Raw json.RawMessage; Devices map[string]DeviceState; Scenarios map[string]string}` (id → имя)
  - `snapshot.Parse(raw json.RawMessage, at time.Time) (Snapshot, error)`
  - `(Snapshot).Value(deviceID, instance string) (v any, ok bool)` — сначала свойство, затем умение.

- [ ] **Step 1: Написать падающий тест**

`internal/snapshot/snapshot_test.go`:

```go
package snapshot

import (
	"encoding/json"
	"testing"
	"time"
)

const sampleUserInfo = `` // вставить образец user/info из раздела плана «Образец user/info для тестов»

var at = time.Date(2026, 9, 8, 12, 0, 0, 0, time.Local)

func TestParseBuildsDeviceStates(t *testing.T) {
	s, err := Parse(json.RawMessage(sampleUserInfo), at)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !s.At.Equal(at) || string(s.Raw) != sampleUserInfo {
		t.Error("At/Raw не сохранены")
	}
	if len(s.Devices) != 5 {
		t.Fatalf("devices = %d, want 5", len(s.Devices))
	}
	d := s.Devices["sensor-office"]
	if d.Name != "Датчик климата" || d.Room != "Кабинет" || d.Type != "devices.types.sensor.climate" {
		t.Errorf("sensor = %+v", d)
	}
	if v, ok := s.Value("sensor-office", "temperature"); !ok || v != 26.3 {
		t.Errorf("temperature = %v, %v", v, ok)
	}
	b := s.Devices["blinds"]
	if v, ok := s.Value("blinds", "open"); !ok || v != float64(100) {
		t.Errorf("open = %v, %v", v, ok)
	}
	if b.CapTypes["open"] != "devices.capabilities.range" || !b.Ranges["open"].HasRange || b.Ranges["open"].Max != 100 {
		t.Errorf("blinds meta = %+v", b)
	}
	f := s.Devices["fan"]
	if got := f.Modes["fan_speed"]; len(got) != 3 || got[1] != "medium" {
		t.Errorf("fan modes = %v", got)
	}
	if v, ok := s.Value("fan", "on"); !ok || v != false {
		t.Errorf("fan on = %v, %v", v, ok)
	}
	if _, ok := s.Value("tv", "on"); ok {
		t.Error("state:null должен давать ok=false")
	}
	if _, ok := s.Value("nope", "on"); ok {
		t.Error("неизвестное устройство должно давать ok=false")
	}
	if s.Scenarios["s1"] != "Я дома" {
		t.Errorf("scenarios = %v", s.Scenarios)
	}
}

func TestParseRejectsBadJSON(t *testing.T) {
	if _, err := Parse(json.RawMessage(`nope`), at); err == nil {
		t.Fatal("ожидалась ошибка")
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/snapshot/`
Expected: ошибка компиляции — `undefined: Parse`.

- [ ] **Step 3: Реализация**

`internal/snapshot/snapshot.go`:

```go
// Package snapshot превращает ответ user/info Яндекса в снимок состояния дома,
// с которым работают движок правил и обработчики API.
package snapshot

import (
	"encoding/json"
	"time"
)

type Range struct {
	Min, Max float64
	HasRange bool
}

type DeviceState struct {
	ID, Name, Type, Room string
	Props                map[string]any      // instance → значение свойства
	Caps                 map[string]any      // instance → состояние умения
	CapTypes             map[string]string   // instance → тип умения (devices.capabilities.*)
	Modes                map[string][]string // instance → допустимые режимы
	Ranges               map[string]Range    // instance → диапазон
}

type Snapshot struct {
	At        time.Time
	Raw       json.RawMessage
	Devices   map[string]DeviceState
	Scenarios map[string]string
}

type rawState struct {
	Instance string `json:"instance"`
	Value    any    `json:"value"`
}

type rawUserInfo struct {
	Rooms []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"rooms"`
	Devices []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Type         string `json:"type"`
		Room         string `json:"room"`
		Capabilities []struct {
			Type       string `json:"type"`
			Parameters struct {
				Instance string `json:"instance"`
				Range    *struct {
					Min float64 `json:"min"`
					Max float64 `json:"max"`
				} `json:"range"`
				Modes []struct {
					Value string `json:"value"`
				} `json:"modes"`
				ColorModel   string `json:"color_model"`
				TemperatureK *struct {
					Min float64 `json:"min"`
					Max float64 `json:"max"`
				} `json:"temperature_k"`
			} `json:"parameters"`
			State *rawState `json:"state"`
		} `json:"capabilities"`
		Properties []struct {
			Parameters struct {
				Instance string `json:"instance"`
			} `json:"parameters"`
			State *rawState `json:"state"`
		} `json:"properties"`
	} `json:"devices"`
	Scenarios []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"scenarios"`
}

func Parse(raw json.RawMessage, at time.Time) (Snapshot, error) {
	var info rawUserInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return Snapshot{}, err
	}
	rooms := map[string]string{}
	for _, r := range info.Rooms {
		rooms[r.ID] = r.Name
	}
	s := Snapshot{At: at, Raw: raw, Devices: map[string]DeviceState{}, Scenarios: map[string]string{}}
	for _, d := range info.Devices {
		st := DeviceState{ID: d.ID, Name: d.Name, Type: d.Type, Room: rooms[d.Room],
			Props: map[string]any{}, Caps: map[string]any{}, CapTypes: map[string]string{},
			Modes: map[string][]string{}, Ranges: map[string]Range{}}
		for _, p := range d.Properties {
			if p.State != nil {
				st.Props[p.Parameters.Instance] = p.State.Value
			}
		}
		for _, c := range d.Capabilities {
			inst := c.Parameters.Instance
			switch c.Type {
			case "devices.capabilities.on_off":
				inst = "on"
			case "devices.capabilities.color_setting":
				// color_setting: сразу несколько instance; метаданные по каждому
				if c.Parameters.TemperatureK != nil {
					st.CapTypes["temperature_k"] = c.Type
					st.Ranges["temperature_k"] = Range{Min: c.Parameters.TemperatureK.Min, Max: c.Parameters.TemperatureK.Max, HasRange: true}
				}
				if c.Parameters.ColorModel != "" {
					st.CapTypes[c.Parameters.ColorModel] = c.Type
				}
				if c.State != nil {
					st.Caps[c.State.Instance] = c.State.Value
				}
				continue
			}
			st.CapTypes[inst] = c.Type
			if c.Parameters.Range != nil {
				st.Ranges[inst] = Range{Min: c.Parameters.Range.Min, Max: c.Parameters.Range.Max, HasRange: true}
			}
			if len(c.Parameters.Modes) > 0 {
				modes := make([]string, 0, len(c.Parameters.Modes))
				for _, m := range c.Parameters.Modes {
					modes = append(modes, m.Value)
				}
				st.Modes[inst] = modes
			}
			if c.State != nil {
				st.Caps[c.State.Instance] = c.State.Value
			}
		}
		s.Devices[d.ID] = st
	}
	for _, sc := range info.Scenarios {
		s.Scenarios[sc.ID] = sc.Name
	}
	return s, nil
}

// Value возвращает текущее значение свойства (приоритет) или умения по instance.
func (s Snapshot) Value(deviceID, instance string) (any, bool) {
	d, ok := s.Devices[deviceID]
	if !ok {
		return nil, false
	}
	if v, ok := d.Props[instance]; ok {
		return v, true
	}
	if v, ok := d.Caps[instance]; ok {
		return v, true
	}
	return nil, false
}
```

- [ ] **Step 4: Проверить**

Run: `gofmt -l . && go vet ./... && go test ./internal/snapshot/ -v`
Expected: 2 теста PASS.

- [ ] **Step 5: Коммит**

```bash
git add internal/snapshot
git commit -m "feat: снимок состояния дома из user/info"
```

---

### Task 2: Опрос дома на сервере

**Files:**
- Create: `internal/poller/poller.go`
- Test: `internal/poller/poller_test.go`

**Interfaces:**
- Consumes: `snapshot.Parse`, `snapshot.Snapshot` (Task 1); интерфейс источника `UserInfo(ctx) (json.RawMessage, error)` (его реализует `*yandex.Client`).
- Produces:
  - `poller.Source` interface `{ UserInfo(ctx) (json.RawMessage, error) }`
  - `poller.New(src Source, interval time.Duration, log *slog.Logger) *Poller`
  - `(*Poller).Run(ctx)` — блокирующий цикл; `(*Poller).Latest() (snapshot.Snapshot, bool)`; `(*Poller).Refresh(ctx) (snapshot.Snapshot, error)` — немедленный опрос (используется API после действий и при пустом снимке); `(*Poller).Subscribe(fn func(snapshot.Snapshot))` — вызывается после каждого успешного опроса (в горутине цикла, последовательно).
  - Ошибки опроса логируются (`Warn`), последний снимок сохраняется; при `auth.ErrNoToken`/401 цикл продолжает попытки с тем же интервалом (сервер сам разлогинивает, а после входа опрос «оживёт»).

- [ ] **Step 1: Написать падающий тест**

`internal/poller/poller_test.go`:

```go
package poller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const sampleUserInfo = `` // тот же образец user/info из плана

type fakeSource struct {
	mu    sync.Mutex
	calls int32
	err   error
	body  string
}

func (f *fakeSource) UserInfo(context.Context) (json.RawMessage, error) {
	atomic.AddInt32(&f.calls, 1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return json.RawMessage(f.body), nil
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestRunPollsAndNotifies(t *testing.T) {
	src := &fakeSource{body: sampleUserInfo}
	p := New(src, 20*time.Millisecond, quiet())
	notified := make(chan int, 16)
	p.Subscribe(func(s snapshot.Snapshot) { notified <- len(s.Devices) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Run(ctx)

	select {
	case n := <-notified:
		if n != 5 {
			t.Fatalf("devices in snapshot = %d, want 5", n)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("подписчик не получил снимок")
	}
	s, ok := p.Latest()
	if !ok || s.Devices["fan"].Name != "Вентилятор" {
		t.Errorf("Latest = %+v, %v", s.Devices["fan"], ok)
	}
	time.Sleep(70 * time.Millisecond)
	if atomic.LoadInt32(&src.calls) < 3 {
		t.Errorf("calls = %d, want ≥3 (опрос по интервалу)", src.calls)
	}
}

func TestErrorsKeepLastSnapshot(t *testing.T) {
	src := &fakeSource{body: sampleUserInfo}
	p := New(src, 20*time.Millisecond, quiet())
	if _, err := p.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	src.mu.Lock()
	src.err = errors.New("yandex api: HTTP 401")
	src.mu.Unlock()
	if _, err := p.Refresh(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка")
	}
	if s, ok := p.Latest(); !ok || len(s.Devices) != 5 {
		t.Errorf("после ошибки снимок должен сохраниться: ok=%v devices=%d", ok, len(s.Devices))
	}
}

func TestLatestEmptyBeforeFirstPoll(t *testing.T) {
	p := New(&fakeSource{body: sampleUserInfo}, time.Second, quiet())
	if _, ok := p.Latest(); ok {
		t.Fatal("до первого опроса снимка быть не должно")
	}
}
```

В импорты файла добавить `"smarthome/internal/snapshot"` (тесты ссылаются на `snapshot.Snapshot`).

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/poller/`
Expected: ошибка компиляции — `undefined: New`.

- [ ] **Step 3: Реализация**

`internal/poller/poller.go`:

```go
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
```

- [ ] **Step 4: Проверить**

Run: `gofmt -l . && go vet ./... && go test ./internal/poller/ -race -v`
Expected: 3 теста PASS, гонок нет.

- [ ] **Step 5: Коммит**

```bash
git add internal/poller
git commit -m "feat: опрос состояния дома на сервере"
```

### Task 3: Модель правил и хранилище

**Files:**
- Create: `internal/rules/rule.go`
- Create: `internal/rules/store.go`
- Test: `internal/rules/store_test.go`

**Interfaces:**
- Consumes: `snapshot.Snapshot`, `snapshot.DeviceState` (Task 1).
- Produces:
  - `rules.TimeWindow{After, Before string}` (`"HH:MM"`), `rules.Condition{DeviceID, Property, Capability, Op string; Value any; Time *TimeWindow}` (json: `device_id`, `property`, `capability`, `op`, `value`, `time`), `rules.Action{DeviceID, Type, Instance string; Value any; MacroID, ScenarioID string}` (json: `device_id`, `type`, `instance`, `value`, `macro_id`, `scenario_id`), `rules.Rule{ID, Name string; Enabled bool; When []Condition; ForMinutes, CooldownMinutes int; Then []Action}` (json: `id`, `name`, `enabled`, `when`, `for_minutes`, `cooldown_minutes`, `then`).
  - `rules.ErrNotFound`; `rules.DefaultCooldownMinutes = 30`.
  - `(Rule).Validate(snap *snapshot.Snapshot, macroExists func(string) bool) error` — `snap == nil` пропускает проверки существования.
  - `rules.NewStore(path) (*Store, error)`; `(*Store).List() []Rule`, `Get(id) (Rule, bool)`, `Create(r Rule, snap *snapshot.Snapshot, macroExists) (Rule, error)`, `Update(r Rule, snap, macroExists) error`, `Delete(id) error`. Create/Update выставляют `CooldownMinutes = 30`, если 0, и `Enabled = true` для новых правил, если поле не задано (используйте `*bool` в промежуточной структуре? — нет: считать, что клиент передаёт `enabled`; при создании через API отсутствующее поле → `true`, это делает обработчик в Task 5, а не Store).

- [ ] **Step 1: Написать падающие тесты**

`internal/rules/store_test.go`:

```go
package rules

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"smarthome/internal/snapshot"
)

const sampleUserInfo = `` // образец user/info из плана

func snap(t *testing.T) *snapshot.Snapshot {
	t.Helper()
	s, err := snapshot.Parse(json.RawMessage(sampleUserInfo), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return &s
}

func officeHot() Rule {
	return Rule{Name: "Кабинет жарко", Enabled: true,
		When: []Condition{
			{DeviceID: "sensor-office", Property: "temperature", Op: ">", Value: 25},
			{DeviceID: "blinds", Capability: "open", Op: ">", Value: 50},
			{Time: &TimeWindow{After: "11:00", Before: "17:00"}},
		},
		Then: []Action{{DeviceID: "blinds", Type: "devices.capabilities.range", Instance: "open", Value: 40}},
	}
}

func TestCreateDefaultsAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rules.json")
	st, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	r, err := st.Create(officeHot(), snap(t), func(string) bool { return false })
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.ID == "" || r.CooldownMinutes != DefaultCooldownMinutes {
		t.Errorf("created = %+v", r)
	}
	re, _ := NewStore(path)
	got, ok := re.Get(r.ID)
	if !ok || got.Name != "Кабинет жарко" || len(got.When) != 3 || got.When[2].Time.After != "11:00" {
		t.Errorf("reloaded = %+v ok=%v", got, ok)
	}
	if got.When[0].Value != float64(25) {
		t.Errorf("value after JSON = %#v, want float64(25)", got.When[0].Value)
	}
}

func TestValidateRejectsBadRules(t *testing.T) {
	s := snap(t)
	noMacro := func(string) bool { return false }
	cases := map[string]Rule{
		"без имени":            {When: officeHot().When, Then: officeHot().Then},
		"без условий":          {Name: "x", Then: officeHot().Then},
		"без действий":         {Name: "x", When: officeHot().When},
		"неизвестное устройство": {Name: "x", When: []Condition{{DeviceID: "nope", Property: "temperature", Op: ">", Value: 1}}, Then: officeHot().Then},
		"неизвестный instance": {Name: "x", When: []Condition{{DeviceID: "sensor-office", Property: "co2", Op: ">", Value: 1}}, Then: officeHot().Then},
		"плохой оператор":      {Name: "x", When: []Condition{{DeviceID: "sensor-office", Property: "temperature", Op: "~", Value: 1}}, Then: officeHot().Then},
		"плохое время":         {Name: "x", When: []Condition{{Time: &TimeWindow{After: "25:00", Before: "17:00"}}}, Then: officeHot().Then},
		"условие без вида":     {Name: "x", When: []Condition{{Op: ">", Value: 1}}, Then: officeHot().Then},
		"режим вне списка":     {Name: "x", When: officeHot().When, Then: []Action{{DeviceID: "fan", Type: "devices.capabilities.mode", Instance: "fan_speed", Value: "turbo"}}},
		"число вне диапазона":  {Name: "x", When: officeHot().When, Then: []Action{{DeviceID: "blinds", Type: "devices.capabilities.range", Instance: "open", Value: 140}}},
		"неизвестный макрос":   {Name: "x", When: officeHot().When, Then: []Action{{MacroID: "m1"}}},
		"неизвестный сценарий": {Name: "x", When: officeHot().When, Then: []Action{{ScenarioID: "s9"}}},
	}
	for name, r := range cases {
		if err := r.Validate(s, noMacro); err == nil {
			t.Errorf("%s: ожидалась ошибка", name)
		}
	}
	ok := officeHot()
	ok.Then = append(ok.Then, Action{MacroID: "m1"}, Action{ScenarioID: "s1"})
	if err := ok.Validate(s, func(id string) bool { return id == "m1" }); err != nil {
		t.Errorf("валидное правило отвергнуто: %v", err)
	}
	if err := officeHot().Validate(nil, noMacro); err != nil {
		t.Errorf("без снимка проверки существования должны пропускаться: %v", err)
	}
}

func TestUpdateDeleteNotFound(t *testing.T) {
	st, _ := NewStore(filepath.Join(t.TempDir(), "rules.json"))
	r, _ := st.Create(officeHot(), nil, nil)
	r.Name = "Кабинет жарко (v2)"
	r.Enabled = false
	if err := st.Update(r, nil, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := st.Get(r.ID)
	if got.Name != "Кабинет жарко (v2)" || got.Enabled {
		t.Errorf("after update = %+v", got)
	}
	bad := r
	bad.ID = "nope"
	if err := st.Update(bad, nil, nil); !errors.Is(err, ErrNotFound) {
		t.Errorf("update unknown: %v", err)
	}
	if err := st.Delete(r.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.Delete(r.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if got := st.List(); got == nil || len(got) != 0 {
		t.Errorf("List = %#v, want пустой срез", got)
	}
}

func TestValidateMessagesAreRussian(t *testing.T) {
	err := Rule{Name: "x", When: []Condition{{DeviceID: "nope", Property: "t", Op: ">", Value: 1}}, Then: officeHot().Then}.Validate(snap(t), nil)
	if err == nil || !strings.Contains(err.Error(), "устройство") {
		t.Errorf("err = %v", err)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/rules/`
Expected: ошибка компиляции — `undefined: Rule`.

- [ ] **Step 3: Реализация модели и валидации**

`internal/rules/rule.go`:

```go
// Package rules — локальные автоматизации: правила «условия → действия»,
// проверяемые на каждом снимке состояния дома.
package rules

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"smarthome/internal/snapshot"
)

const DefaultCooldownMinutes = 30

var ErrNotFound = errors.New("правило не найдено")

type TimeWindow struct {
	After  string `json:"after"`
	Before string `json:"before"`
}

type Condition struct {
	DeviceID   string      `json:"device_id,omitempty"`
	Property   string      `json:"property,omitempty"`
	Capability string      `json:"capability,omitempty"`
	Op         string      `json:"op,omitempty"`
	Value      any         `json:"value,omitempty"`
	Time       *TimeWindow `json:"time,omitempty"`
}

type Action struct {
	DeviceID   string `json:"device_id,omitempty"`
	Type       string `json:"type,omitempty"`
	Instance   string `json:"instance,omitempty"`
	Value      any    `json:"value,omitempty"`
	MacroID    string `json:"macro_id,omitempty"`
	ScenarioID string `json:"scenario_id,omitempty"`
}

type Rule struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Enabled         bool        `json:"enabled"`
	When            []Condition `json:"when"`
	ForMinutes      int         `json:"for_minutes"`
	CooldownMinutes int         `json:"cooldown_minutes"`
	Then            []Action    `json:"then"`
}

var validOps = map[string]bool{"<": true, "<=": true, ">": true, ">=": true, "==": true, "!=": true}

// instance условия/действия: свойство или умение.
func (c Condition) instance() string {
	if c.Property != "" {
		return c.Property
	}
	return c.Capability
}

// Validate проверяет форму правила и, если передан снимок, существование
// устройств, instance, режимов и диапазонов. macroExists может быть nil.
func (r Rule) Validate(snap *snapshot.Snapshot, macroExists func(string) bool) error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("у правила должно быть имя")
	}
	if len(r.When) == 0 {
		return errors.New("правило должно содержать хотя бы одно условие")
	}
	if len(r.Then) == 0 {
		return errors.New("правило должно содержать хотя бы одно действие")
	}
	if r.ForMinutes < 0 || r.CooldownMinutes < 0 {
		return errors.New("for_minutes и cooldown_minutes не могут быть отрицательными")
	}
	for i, c := range r.When {
		if err := c.validate(snap); err != nil {
			return fmt.Errorf("условие %d: %w", i+1, err)
		}
	}
	for i, a := range r.Then {
		if err := a.validate(snap, macroExists); err != nil {
			return fmt.Errorf("действие %d: %w", i+1, err)
		}
	}
	return nil
}

func (c Condition) validate(snap *snapshot.Snapshot) error {
	kinds := 0
	if c.Property != "" {
		kinds++
	}
	if c.Capability != "" {
		kinds++
	}
	if c.Time != nil {
		kinds++
	}
	if kinds != 1 {
		return errors.New("нужно ровно одно из: property, capability или time")
	}
	if c.Time != nil {
		if _, err := parseClock(c.Time.After); err != nil {
			return fmt.Errorf("time.after: %w", err)
		}
		if _, err := parseClock(c.Time.Before); err != nil {
			return fmt.Errorf("time.before: %w", err)
		}
		return nil
	}
	if c.DeviceID == "" {
		return errors.New("нужен device_id")
	}
	if !validOps[c.Op] {
		return fmt.Errorf("неизвестный оператор %q (допустимы < <= > >= == !=)", c.Op)
	}
	if c.Value == nil {
		return errors.New("нужно значение value")
	}
	if snap == nil {
		return nil
	}
	d, ok := snap.Devices[c.DeviceID]
	if !ok {
		return fmt.Errorf("устройство %s не найдено в доме", c.DeviceID)
	}
	inst := c.instance()
	if _, ok := d.Props[inst]; ok {
		return nil
	}
	if _, ok := d.Caps[inst]; ok {
		return nil
	}
	if _, ok := d.CapTypes[inst]; ok {
		return nil
	}
	return fmt.Errorf("у устройства «%s» нет показания или умения %q", d.Name, inst)
}

func (a Action) validate(snap *snapshot.Snapshot, macroExists func(string) bool) error {
	switch {
	case a.MacroID != "":
		if macroExists != nil && !macroExists(a.MacroID) {
			return fmt.Errorf("макрос %s не найден", a.MacroID)
		}
		return nil
	case a.ScenarioID != "":
		if snap != nil {
			if _, ok := snap.Scenarios[a.ScenarioID]; !ok {
				return fmt.Errorf("сценарий Яндекса %s не найден", a.ScenarioID)
			}
		}
		return nil
	}
	if a.DeviceID == "" || a.Type == "" || a.Instance == "" {
		return errors.New("нужны device_id, type и instance (или macro_id / scenario_id)")
	}
	if snap == nil {
		return nil
	}
	d, ok := snap.Devices[a.DeviceID]
	if !ok {
		return fmt.Errorf("устройство %s не найдено в доме", a.DeviceID)
	}
	if modes, ok := d.Modes[a.Instance]; ok {
		s, isStr := a.Value.(string)
		found := false
		for _, m := range modes {
			if isStr && m == s {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("у «%s» нет режима %v (доступны: %s)", d.Name, a.Value, strings.Join(modes, ", "))
		}
	}
	if rg, ok := d.Ranges[a.Instance]; ok && rg.HasRange {
		f, isNum := toFloat(a.Value)
		if !isNum || f < rg.Min || f > rg.Max {
			return fmt.Errorf("значение %v для «%s» вне диапазона %g..%g", a.Value, d.Name, rg.Min, rg.Max)
		}
	}
	return nil
}

// parseClock разбирает "HH:MM" в минуты от полуночи.
func parseClock(s string) (int, error) {
	hh, mm, ok := strings.Cut(s, ":")
	if !ok {
		return 0, fmt.Errorf("ожидается ЧЧ:ММ, получено %q", s)
	}
	h, err1 := strconv.Atoi(hh)
	m, err2 := strconv.Atoi(mm)
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("ожидается ЧЧ:ММ, получено %q", s)
	}
	return h*60 + m, nil
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}
```

Добавить в импорты `"encoding/json"` (для `json.Number`).

- [ ] **Step 4: Реализация хранилища**

`internal/rules/store.go`:

```go
package rules

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"smarthome/internal/snapshot"
)

// Store хранит правила в JSON-файле; сервер — единственный писатель.
type Store struct {
	path string

	mu    sync.Mutex
	rules []Rule
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &s.rules); err != nil {
		return nil, fmt.Errorf("файл правил %s: %w", path, err)
	}
	return s, nil
}

func (s *Store) List() []Rule {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Rule, len(s.rules))
	copy(out, s.rules)
	return out
}

func (s *Store) Get(id string) (Rule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i := s.index(id); i >= 0 {
		return s.rules[i], true
	}
	return Rule{}, false
}

func (s *Store) Create(r Rule, snap *snapshot.Snapshot, macroExists func(string) bool) (Rule, error) {
	if r.CooldownMinutes == 0 {
		r.CooldownMinutes = DefaultCooldownMinutes
	}
	if err := r.Validate(snap, macroExists); err != nil {
		return Rule{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r.ID = newID()
	s.rules = append(s.rules, r)
	if err := s.save(); err != nil {
		s.rules = s.rules[:len(s.rules)-1]
		return Rule{}, err
	}
	return r, nil
}

func (s *Store) Update(r Rule, snap *snapshot.Snapshot, macroExists func(string) bool) error {
	if r.CooldownMinutes == 0 {
		r.CooldownMinutes = DefaultCooldownMinutes
	}
	if err := r.Validate(snap, macroExists); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(r.ID)
	if i < 0 {
		return ErrNotFound
	}
	old := s.rules[i]
	s.rules[i] = r
	if err := s.save(); err != nil {
		s.rules[i] = old
		return err
	}
	return nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(id)
	if i < 0 {
		return ErrNotFound
	}
	old := s.rules
	s.rules = append(s.rules[:i:i], s.rules[i+1:]...)
	if err := s.save(); err != nil {
		s.rules = old
		return err
	}
	return nil
}

func (s *Store) index(id string) int {
	for i, r := range s.rules {
		if r.ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.rules, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand недоступен: " + err.Error())
	}
	return hex.EncodeToString(b)
}
```

- [ ] **Step 5: Проверить**

Run: `gofmt -l . && go vet ./... && go test ./internal/rules/ -v`
Expected: 4 теста PASS.

- [ ] **Step 6: Коммит**

```bash
git add internal/rules
git commit -m "feat: модель и хранилище правил автоматизации"
```

---

### Task 4: Движок правил

**Files:**
- Create: `internal/rules/engine.go`
- Test: `internal/rules/engine_test.go`

**Interfaces:**
- Consumes: `Rule`, `Condition`, `TimeWindow`, `parseClock`, `toFloat` (Task 3); `snapshot.Snapshot.Value` (Task 1).
- Produces:
  - `rules.CondResult{Index int; OK bool; Current any; Kind string}` (Kind: `property` | `capability` | `time`) и `rules.Evaluate(r Rule, snap snapshot.Snapshot, now time.Time) (bool, []CondResult)`.
  - `rules.NewEngine(now func() time.Time) *Engine`; `(*Engine).Tick(all []Rule, snap snapshot.Snapshot) []Rule` — возвращает правила, которые сработали на этом снимке; `(*Engine).LastFired(id) (time.Time, bool)`.
  - Семантика: условие истинно N ≥ `for_minutes` минут подряд → срабатывание; после срабатывания правило «взведётся» снова только когда условия станут ложными; если фронт наступил во время кулдауна, правило остаётся взведённым и сработает по окончании кулдауна, пока условия истинны; выключенное правило сбрасывает состояние.

- [ ] **Step 1: Написать падающие тесты**

`internal/rules/engine_test.go`:

```go
package rules

import (
	"encoding/json"
	"testing"
	"time"

	"smarthome/internal/snapshot"
)

// snapWith возвращает снимок образца с подменённой температурой в кабинете и положением жалюзи.
func snapWith(t *testing.T, temp float64, open float64, at time.Time) snapshot.Snapshot {
	t.Helper()
	s, err := snapshot.Parse(json.RawMessage(sampleUserInfo), at)
	if err != nil {
		t.Fatal(err)
	}
	s.Devices["sensor-office"].Props["temperature"] = temp
	s.Devices["blinds"].Caps["open"] = open
	return s
}

func at(h, m int) time.Time { return time.Date(2026, 9, 8, h, m, 0, 0, time.Local) }

func TestEvaluateConditions(t *testing.T) {
	r := officeHot()
	ok, res := Evaluate(r, snapWith(t, 26.3, 100, at(12, 0)), at(12, 0))
	if !ok || len(res) != 3 || !res[0].OK || !res[1].OK || !res[2].OK || res[0].Current != 26.3 || res[0].Kind != "property" {
		t.Errorf("ok=%v res=%+v", ok, res)
	}
	if ok, res := Evaluate(r, snapWith(t, 24, 100, at(12, 0)), at(12, 0)); ok || res[0].OK {
		t.Error("температура 24 не должна проходить > 25")
	}
	if ok, res := Evaluate(r, snapWith(t, 26.3, 40, at(12, 0)), at(12, 0)); ok || res[1].OK {
		t.Error("жалюзи 40 не должны проходить > 50")
	}
	if ok, res := Evaluate(r, snapWith(t, 26.3, 100, at(18, 0)), at(18, 0)); ok || res[2].OK {
		t.Error("18:00 вне окна 11:00–17:00")
	}
}

func TestEvaluateOperatorsAndTypes(t *testing.T) {
	s := snapWith(t, 26.3, 100, at(12, 0))
	cases := []struct {
		c    Condition
		want bool
	}{
		{Condition{DeviceID: "fan", Capability: "on", Op: "==", Value: false}, true},
		{Condition{DeviceID: "fan", Capability: "on", Op: "!=", Value: false}, false},
		{Condition{DeviceID: "fan", Capability: "fan_speed", Op: "==", Value: "low"}, true},
		{Condition{DeviceID: "sensor-office", Property: "temperature", Op: ">=", Value: 26.3}, true},
		{Condition{DeviceID: "sensor-office", Property: "temperature", Op: "<=", Value: 26}, false},
		{Condition{DeviceID: "sensor-office", Property: "temperature", Op: "<", Value: 30}, true},
		{Condition{DeviceID: "tv", Capability: "on", Op: "==", Value: true}, false}, // state:null → неизвестно → false
		{Condition{DeviceID: "nope", Property: "x", Op: "==", Value: 1}, false},
	}
	for i, tc := range cases {
		ok, _ := Evaluate(Rule{When: []Condition{tc.c}}, s, at(12, 0))
		if ok != tc.want {
			t.Errorf("case %d (%+v): ok=%v want %v", i, tc.c, ok, tc.want)
		}
	}
}

func TestTimeWindowOvernight(t *testing.T) {
	r := Rule{When: []Condition{{Time: &TimeWindow{After: "22:00", Before: "08:00"}}}}
	s := snapWith(t, 20, 0, at(0, 0))
	for _, tc := range []struct {
		now  time.Time
		want bool
	}{{at(23, 30), true}, {at(2, 0), true}, {at(7, 59), true}, {at(8, 0), false}, {at(12, 0), false}, {at(22, 0), true}} {
		if ok, _ := Evaluate(r, s, tc.now); ok != tc.want {
			t.Errorf("%s: ok=%v want %v", tc.now.Format("15:04"), ok, tc.want)
		}
	}
}

func TestTickEdgeForMinutesCooldown(t *testing.T) {
	now := at(12, 0)
	e := NewEngine(func() time.Time { return now })
	r := officeHot()
	r.ID = "r1"
	r.ForMinutes = 5
	r.CooldownMinutes = 60
	hot := func(min int) snapshot.Snapshot { return snapWith(t, 26.3, 100, at(12, min)) }
	cold := func(min int) snapshot.Snapshot { return snapWith(t, 22, 100, at(12, min)) }

	if fired := e.Tick([]Rule{r}, hot(0)); len(fired) != 0 {
		t.Fatal("не должно срабатывать раньше for_minutes")
	}
	now = at(12, 4)
	if fired := e.Tick([]Rule{r}, hot(4)); len(fired) != 0 {
		t.Fatal("4 минуты < 5")
	}
	now = at(12, 5)
	if fired := e.Tick([]Rule{r}, hot(5)); len(fired) != 1 || fired[0].ID != "r1" {
		t.Fatalf("должно сработать на 5-й минуте: %+v", fired)
	}
	if lf, ok := e.LastFired("r1"); !ok || !lf.Equal(at(12, 5)) {
		t.Errorf("LastFired = %v %v", lf, ok)
	}
	now = at(12, 6)
	if fired := e.Tick([]Rule{r}, hot(6)); len(fired) != 0 {
		t.Fatal("без сброса условий повторного срабатывания быть не должно")
	}
	// Сброс и новый фронт во время кулдауна: остаёмся взведёнными, срабатываем по окончании кулдауна.
	now = at(12, 10)
	e.Tick([]Rule{r}, cold(10))
	now = at(12, 20)
	if fired := e.Tick([]Rule{r}, hot(20)); len(fired) != 0 {
		t.Fatal("кулдаун 60 минут ещё не истёк")
	}
	now = at(13, 6)
	if fired := e.Tick([]Rule{r}, hot(66)); len(fired) != 1 {
		t.Fatal("после кулдауна при истинных условиях должно сработать")
	}
	// Выключенное правило не срабатывает и сбрасывает состояние.
	r.Enabled = false
	now = at(15, 0)
	e.Tick([]Rule{r}, cold(0))
	r.Enabled = true
	if fired := e.Tick([]Rule{r}, hot(0)); len(fired) != 0 {
		t.Fatal("после включения нужен for_minutes заново")
	}
}

func TestTickZeroForMinutesFiresImmediately(t *testing.T) {
	now := at(12, 0)
	e := NewEngine(func() time.Time { return now })
	r := officeHot()
	r.ID = "r2"
	if fired := e.Tick([]Rule{r}, snapWith(t, 26.3, 100, now)); len(fired) != 1 {
		t.Fatalf("for_minutes=0 должно срабатывать на первом истинном снимке: %+v", fired)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/rules/`
Expected: ошибка компиляции — `undefined: Evaluate`, `undefined: NewEngine`.

- [ ] **Step 3: Реализация**

`internal/rules/engine.go`:

```go
package rules

import (
	"sync"
	"time"

	"smarthome/internal/snapshot"
)

type CondResult struct {
	Index   int    `json:"index"`
	OK      bool   `json:"ok"`
	Current any    `json:"current,omitempty"`
	Kind    string `json:"kind"`
}

// Evaluate вычисляет все условия правила на снимке; ok — все истинны.
func Evaluate(r Rule, snap snapshot.Snapshot, now time.Time) (bool, []CondResult) {
	all := true
	res := make([]CondResult, 0, len(r.When))
	for i, c := range r.When {
		cr := CondResult{Index: i}
		switch {
		case c.Time != nil:
			cr.Kind = "time"
			cr.Current = now.Format("15:04")
			cr.OK = inWindow(*c.Time, now)
		default:
			if c.Property != "" {
				cr.Kind = "property"
			} else {
				cr.Kind = "capability"
			}
			cur, ok := snap.Value(c.DeviceID, c.instance())
			cr.Current = cur
			cr.OK = ok && compare(cur, c.Op, c.Value)
		}
		if !cr.OK {
			all = false
		}
		res = append(res, cr)
	}
	return all, res
}

func inWindow(w TimeWindow, now time.Time) bool {
	after, err1 := parseClock(w.After)
	before, err2 := parseClock(w.Before)
	if err1 != nil || err2 != nil {
		return false
	}
	cur := now.Hour()*60 + now.Minute()
	if after <= before {
		return cur >= after && cur < before
	}
	return cur >= after || cur < before // через полночь
}

func compare(cur any, op string, want any) bool {
	if cf, ok1 := toFloat(cur); ok1 {
		wf, ok2 := toFloat(want)
		if !ok2 {
			return false
		}
		switch op {
		case "<":
			return cf < wf
		case "<=":
			return cf <= wf
		case ">":
			return cf > wf
		case ">=":
			return cf >= wf
		case "==":
			return cf == wf
		case "!=":
			return cf != wf
		}
		return false
	}
	switch op {
	case "==":
		return cur == want
	case "!=":
		return cur != want
	}
	return false
}

type ruleState struct {
	trueSince time.Time
	armed     bool
	lastFired time.Time
}

// Engine хранит состояние срабатывания правил между снимками.
type Engine struct {
	now   func() time.Time
	mu    sync.Mutex
	state map[string]*ruleState
}

func NewEngine(now func() time.Time) *Engine {
	if now == nil {
		now = time.Now
	}
	return &Engine{now: now, state: map[string]*ruleState{}}
}

func (e *Engine) LastFired(id string) (time.Time, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	st, ok := e.state[id]
	if !ok || st.lastFired.IsZero() {
		return time.Time{}, false
	}
	return st.lastFired, true
}

// Tick оценивает правила на снимке и возвращает сработавшие.
func (e *Engine) Tick(all []Rule, snap snapshot.Snapshot) []Rule {
	now := e.now()
	e.mu.Lock()
	defer e.mu.Unlock()
	var fired []Rule
	for _, r := range all {
		st, ok := e.state[r.ID]
		if !ok {
			st = &ruleState{armed: true}
			e.state[r.ID] = st
		}
		if !r.Enabled {
			st.trueSince, st.armed = time.Time{}, true
			continue
		}
		ok, _ = Evaluate(r, snap, now)
		if !ok {
			st.trueSince, st.armed = time.Time{}, true
			continue
		}
		if st.trueSince.IsZero() {
			st.trueSince = now
		}
		if !st.armed {
			continue
		}
		if now.Sub(st.trueSince) < time.Duration(r.ForMinutes)*time.Minute {
			continue
		}
		if !st.lastFired.IsZero() && now.Sub(st.lastFired) < time.Duration(r.CooldownMinutes)*time.Minute {
			continue
		}
		st.lastFired, st.armed = now, false
		fired = append(fired, r)
	}
	return fired
}
```

- [ ] **Step 4: Проверить**

Run: `gofmt -l . && go vet ./... && go test ./internal/rules/ -v`
Expected: все тесты PASS (4 из Task 3 + 5 новых).

- [ ] **Step 5: Коммит**

```bash
git add internal/rules
git commit -m "feat: движок правил — условия, окно времени, фронт и кулдаун"
```

### Task 5: Журнал событий и исполнение действий

**Files:**
- Create: `internal/rules/events.go`
- Create: `internal/rules/runner.go`
- Test: `internal/rules/runner_test.go`

**Interfaces:**
- Consumes: `Rule`, `Action` (Task 3); `yandex.ActionsRequest/DeviceActions/Action/ActionState`; `macros.Store.Get`, `macros.Action`.
- Produces:
  - `rules.Event{Time time.Time; RuleID, RuleName string; OK bool; Details string}` (json: `time`, `rule_id`, `rule_name`, `ok`, `details`).
  - `rules.NewEventLog(path string, keep int) (*EventLog, error)` — читает последние `keep` строк файла, если он есть; `(*EventLog).Append(e Event) error` — дописывает JSON-строку и хранит кольцо из `keep`; `(*EventLog).List(limit int) []Event` — новые сверху.
  - `rules.Home` interface `{ DeviceActions(ctx, yandex.ActionsRequest) (json.RawMessage, error); RunScenario(ctx, id string) (json.RawMessage, error) }` — реализует `*yandex.Client`.
  - `rules.Runner{Home Home; Macros *macros.Store}`; `(*Runner).Execute(ctx, r Rule) (details string, err error)` — собирает все действия над устройствами (включая развёрнутые макросы) в один запрос `devices/actions`, затем запускает сценарии по одному; `details` — «устройств: N, сценариев: M» плюс перечень ошибок `action_result` по устройствам; `err != nil`, если запрос не удался или хотя бы одно действие вернуло статус ≠ DONE.

- [ ] **Step 1: Написать падающие тесты**

`internal/rules/runner_test.go`:

```go
package rules

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"smarthome/internal/macros"
	"smarthome/internal/yandex"
)

type fakeHome struct {
	actions   *yandex.ActionsRequest
	resp      string
	err       error
	scenarios []string
}

func (f *fakeHome) DeviceActions(_ context.Context, req yandex.ActionsRequest) (json.RawMessage, error) {
	f.actions = &req
	if f.err != nil {
		return nil, f.err
	}
	return json.RawMessage(f.resp), nil
}

func (f *fakeHome) RunScenario(_ context.Context, id string) (json.RawMessage, error) {
	f.scenarios = append(f.scenarios, id)
	return json.RawMessage(`{"status":"ok","request_id":"s"}`), f.err
}

const okResp = `{"status":"ok","request_id":"r","devices":[{"id":"blinds","capabilities":[{"type":"devices.capabilities.range","state":{"instance":"open","action_result":{"status":"DONE"}}}]}]}`
const failResp = `{"status":"ok","request_id":"r","devices":[{"id":"blinds","capabilities":[{"type":"devices.capabilities.range","state":{"instance":"open","action_result":{"status":"ERROR","error_code":"DEVICE_UNREACHABLE","error_message":"Устройство не отвечает"}}}]}]}`

func TestExecuteGroupsDeviceActionsAndMacros(t *testing.T) {
	ms, _ := macros.NewStore(filepath.Join(t.TempDir(), "macros.json"))
	m, _ := ms.Create(macros.Macro{Name: "Свет", Actions: []macros.Action{
		{DeviceID: "fan", Type: "devices.capabilities.on_off", Instance: "on", Value: true},
		{DeviceID: "blinds", Type: "devices.capabilities.on_off", Instance: "on", Value: true},
	}})
	home := &fakeHome{resp: okResp}
	r := &Runner{Home: home, Macros: ms}
	rule := Rule{ID: "r1", Name: "Кабинет жарко", Then: []Action{
		{DeviceID: "blinds", Type: "devices.capabilities.range", Instance: "open", Value: 40},
		{MacroID: m.ID},
		{ScenarioID: "s1"},
	}}

	details, err := r.Execute(context.Background(), rule)
	if err != nil {
		t.Fatalf("Execute: %v (%s)", err, details)
	}
	if home.actions == nil || len(home.actions.Devices) != 2 {
		t.Fatalf("devices in request = %+v, want 2 (blinds, fan)", home.actions)
	}
	var blinds yandex.DeviceActions
	for _, d := range home.actions.Devices {
		if d.ID == "blinds" {
			blinds = d
		}
	}
	if len(blinds.Actions) != 2 {
		t.Errorf("blinds actions = %+v, want range+on_off в одном запросе", blinds.Actions)
	}
	if len(home.scenarios) != 1 || home.scenarios[0] != "s1" {
		t.Errorf("scenarios = %v", home.scenarios)
	}
	if !strings.Contains(details, "устройств: 2") || !strings.Contains(details, "сценариев: 1") {
		t.Errorf("details = %q", details)
	}
}

func TestExecuteReportsActionResultErrors(t *testing.T) {
	home := &fakeHome{resp: failResp}
	r := &Runner{Home: home}
	rule := Rule{ID: "r1", Name: "x", Then: []Action{{DeviceID: "blinds", Type: "devices.capabilities.range", Instance: "open", Value: 40}}}
	details, err := r.Execute(context.Background(), rule)
	if err == nil || !strings.Contains(details, "blinds") || !strings.Contains(details, "не отвечает") {
		t.Errorf("err=%v details=%q", err, details)
	}
}

func TestExecuteTransportError(t *testing.T) {
	r := &Runner{Home: &fakeHome{err: errors.New("yandex api: HTTP 500")}}
	rule := Rule{ID: "r1", Name: "x", Then: []Action{{DeviceID: "blinds", Type: "devices.capabilities.range", Instance: "open", Value: 40}}}
	if _, err := r.Execute(context.Background(), rule); err == nil {
		t.Fatal("ожидалась ошибка")
	}
}

func TestEventLogAppendsAndReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "events.jsonl")
	l, err := NewEventLog(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 4; i++ {
		if err := l.Append(Event{Time: time.Date(2026, 9, 8, 12, i, 0, 0, time.Local), RuleID: "r", RuleName: "Правило", OK: i%2 == 0, Details: "d"}); err != nil {
			t.Fatal(err)
		}
	}
	got := l.List(10)
	if len(got) != 3 || got[0].Time.Minute() != 4 || got[2].Time.Minute() != 2 {
		t.Errorf("List = %+v (ожидались 3 последних, новые сверху)", got)
	}
	if got := l.List(1); len(got) != 1 || got[0].Time.Minute() != 4 {
		t.Errorf("List(1) = %+v", got)
	}
	data, _ := os.ReadFile(path)
	if lines := strings.Count(strings.TrimSpace(string(data)), "\n") + 1; lines != 4 {
		t.Errorf("в файле %d строк, want 4 (файл не усекается)", lines)
	}
	re, err := NewEventLog(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	if got := re.List(10); len(got) != 3 || got[0].Time.Minute() != 4 {
		t.Errorf("после перезагрузки List = %+v", got)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/rules/`
Expected: ошибка компиляции — `undefined: Runner`, `undefined: NewEventLog`.

- [ ] **Step 3: Журнал событий**

`internal/rules/events.go`:

```go
package rules

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Event struct {
	Time     time.Time `json:"time"`
	RuleID   string    `json:"rule_id"`
	RuleName string    `json:"rule_name"`
	OK       bool      `json:"ok"`
	Details  string    `json:"details"`
}

// EventLog дописывает события в JSONL-файл и держит последние keep в памяти.
type EventLog struct {
	path string
	keep int

	mu   sync.Mutex
	ring []Event // старые → новые
}

func NewEventLog(path string, keep int) (*EventLog, error) {
	l := &EventLog{path: path, keep: keep}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var e Event
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			l.push(e)
		}
	}
	return l, sc.Err()
}

func (l *EventLog) push(e Event) {
	l.ring = append(l.ring, e)
	if len(l.ring) > l.keep {
		l.ring = l.ring[len(l.ring)-l.keep:]
	}
}

func (l *EventLog) Append(e Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	l.push(e)
	return nil
}

// List возвращает до limit последних событий, новые сверху.
func (l *EventLog) List(limit int) []Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(l.ring)
	if limit > 0 && limit < n {
		n = limit
	}
	out := make([]Event, 0, n)
	for i := len(l.ring) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, l.ring[i])
	}
	return out
}
```

- [ ] **Step 4: Исполнитель**

`internal/rules/runner.go`:

```go
package rules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"smarthome/internal/macros"
	"smarthome/internal/yandex"
)

type Home interface {
	DeviceActions(ctx context.Context, req yandex.ActionsRequest) (json.RawMessage, error)
	RunScenario(ctx context.Context, id string) (json.RawMessage, error)
}

// Runner выполняет действия правила: устройства — одним запросом, сценарии — по очереди.
type Runner struct {
	Home   Home
	Macros *macros.Store
}

func (r *Runner) Execute(ctx context.Context, rule Rule) (string, error) {
	var acts []macros.Action
	var scenarios []string
	for _, a := range rule.Then {
		switch {
		case a.MacroID != "":
			if r.Macros == nil {
				return "", fmt.Errorf("макросы недоступны")
			}
			m, ok := r.Macros.Get(a.MacroID)
			if !ok {
				return "", fmt.Errorf("макрос %s не найден", a.MacroID)
			}
			acts = append(acts, m.Actions...)
		case a.ScenarioID != "":
			scenarios = append(scenarios, a.ScenarioID)
		default:
			acts = append(acts, macros.Action{DeviceID: a.DeviceID, Type: a.Type, Instance: a.Instance, Value: a.Value})
		}
	}

	var problems []string
	devices := 0
	if len(acts) > 0 {
		req := macros.Macro{Actions: acts}.ToActionsRequest()
		devices = len(req.Devices)
		raw, err := r.Home.DeviceActions(ctx, req)
		if err != nil {
			return fmt.Sprintf("устройств: %d, сценариев: %d", devices, len(scenarios)), err
		}
		problems = append(problems, actionErrors(raw)...)
	}
	for _, id := range scenarios {
		if _, err := r.Home.RunScenario(ctx, id); err != nil {
			problems = append(problems, fmt.Sprintf("сценарий %s: %v", id, err))
		}
	}
	details := fmt.Sprintf("устройств: %d, сценариев: %d", devices, len(scenarios))
	if len(problems) > 0 {
		details += "; ошибки: " + strings.Join(problems, "; ")
		return details, errors.New("часть действий не выполнена")
	}
	return details, nil
}

// actionErrors извлекает из ответа devices/actions действия со статусом ≠ DONE.
func actionErrors(raw json.RawMessage) []string {
	var resp struct {
		Devices []struct {
			ID           string `json:"id"`
			Capabilities []struct {
				State struct {
					Instance     string `json:"instance"`
					ActionResult struct {
						Status       string `json:"status"`
						ErrorCode    string `json:"error_code"`
						ErrorMessage string `json:"error_message"`
					} `json:"action_result"`
				} `json:"state"`
			} `json:"capabilities"`
		} `json:"devices"`
	}
	if json.Unmarshal(raw, &resp) != nil {
		return nil
	}
	var out []string
	for _, d := range resp.Devices {
		for _, c := range d.Capabilities {
			ar := c.State.ActionResult
			if ar.Status != "" && ar.Status != "DONE" {
				msg := ar.ErrorMessage
				if msg == "" {
					msg = ar.ErrorCode
				}
				out = append(out, fmt.Sprintf("%s/%s: %s", d.ID, c.State.Instance, msg))
			}
		}
	}
	return out
}
```

- [ ] **Step 5: Проверить**

Run: `gofmt -l . && go vet ./... && go test ./internal/rules/ -v`
Expected: все тесты PASS.

- [ ] **Step 6: Коммит**

```bash
git add internal/rules
git commit -m "feat: исполнение действий правил и журнал событий"
```

---

### Task 6: API правил, снимок в обработчиках, точка входа

**Files:**
- Modify: `internal/config/config.go` (+ `PollSeconds`, `RulesEnabled`), `internal/config/config_test.go`
- Modify: `internal/server/server.go` (поля, маршруты)
- Modify: `internal/server/home_handlers.go` (`/api/home`, `/api/catalog` из снимка; `Kick` после действий)
- Create: `internal/server/rule_handlers.go`
- Test: `internal/server/rule_handlers_test.go`, дополнить `internal/server/server_test.go`
- Modify: `main.go`

**Interfaces:**
- Consumes: `poller.Poller` (Task 2), `rules.*` (Tasks 3–5), `snapshot.Snapshot`.
- Produces:
  - `config.Config.PollSeconds int` (env `POLL_SECONDS`, по умолчанию 10, минимум 3), `config.Config.RulesEnabled bool` (env `RULES_ENABLED`, по умолчанию `true`; `false`/`0`/`no` выключают).
  - `server.Snapshots` interface `{ Latest() (snapshot.Snapshot, bool); Refresh(ctx) (snapshot.Snapshot, error); Kick() }` — реализует `*poller.Poller`. Поля `Server.Snapshots Snapshots` (nil — старое поведение: ходить в Яндекс напрямую), `Server.Rules *rules.Store`, `Server.Runner *rules.Runner`, `Server.Events *rules.EventLog`, `Server.Engine *rules.Engine` (для `last_fired`).
  - HTTP:

| Запрос | Ответ |
|---|---|
| `GET /api/home` | тело последнего снимка (`Raw`), заголовок `X-Snapshot-Age: <сек>`; при отсутствии снимка — `Refresh` |
| `GET /api/catalog` | `catalog.Build(snapshot.Raw)` |
| `GET /api/rules` | `[{...rule, "last_fired": "RFC3339"?}]` |
| `POST /api/rules` `{name, when, then, for_minutes?, cooldown_minutes?, enabled?}` | 201 правило; `enabled` по умолчанию `true`; 400 `bad_request` с текстом валидации |
| `PUT /api/rules/{id}` | 200 / 404 / 400 |
| `DELETE /api/rules/{id}` | 204 / 404 |
| `POST /api/rules/{id}/run` | 200 `{"ok": bool, "details": "…"}` — выполнить действия немедленно; событие в журнал с пометкой «вручную» |
| `GET /api/rules/{id}/check` | 200 `{"ok": bool, "conditions": [CondResult…], "snapshot_age": сек}`; 404 |
| `GET /api/events?limit=100` | `[Event…]` новые сверху (limit по умолчанию 100, максимум 500) |

- [ ] **Step 1: Конфиг**

Добавить в `internal/config/config_test.go`:

```go
func TestLoadPollAndRules(t *testing.T) {
	clearEnv(t)
	t.Setenv("YANDEX_CLIENT_ID", "id")
	t.Setenv("YANDEX_CLIENT_SECRET", "sec")
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.env"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PollSeconds != 10 || !cfg.RulesEnabled {
		t.Errorf("defaults: poll=%d rules=%v", cfg.PollSeconds, cfg.RulesEnabled)
	}
	t.Setenv("POLL_SECONDS", "1")
	t.Setenv("RULES_ENABLED", "false")
	cfg, _ = Load(filepath.Join(t.TempDir(), "nope.env"))
	if cfg.PollSeconds != 3 || cfg.RulesEnabled {
		t.Errorf("overrides: poll=%d (min 3) rules=%v", cfg.PollSeconds, cfg.RulesEnabled)
	}
}
```

(в `clearEnv` добавить `POLL_SECONDS`, `RULES_ENABLED`). В `config.go`: поля `PollSeconds int`, `RulesEnabled bool`; разбор — `strconv.Atoi(get("POLL_SECONDS","10"))` (ошибка разбора → ошибка Load с русским текстом; значение < 3 → 3); `RulesEnabled = !slices.Contains([]string{"false","0","no","off"}, strings.ToLower(get("RULES_ENABLED","true")))`. Обновить `.env.example` (`POLL_SECONDS=10`, `RULES_ENABLED=true` с комментарием «на ноутбуке разработчика — false, чтобы правила выполнял только домашний сервер»).

- [ ] **Step 2: Тесты обработчиков**

`internal/server/rule_handlers_test.go` (использует `fakeHome`, `fakeAuth`, `do`, `decode` из `server_test.go`; добавить в `server_test.go` образец `sampleUserInfo` из плана и фейк снимков):

```go
package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"smarthome/internal/macros"
	"smarthome/internal/rules"
	"smarthome/internal/snapshot"
)

type fakeSnapshots struct {
	snap  snapshot.Snapshot
	has   bool
	kicks int
}

func (f *fakeSnapshots) Latest() (snapshot.Snapshot, bool) { return f.snap, f.has }
func (f *fakeSnapshots) Refresh(context.Context) (snapshot.Snapshot, error) {
	f.has = true
	return f.snap, nil
}
func (f *fakeSnapshots) Kick() { f.kicks++ }

func newRulesServer(t *testing.T, home *fakeHome) (*httptest.Server, *fakeSnapshots, *rules.Store) {
	t.Helper()
	snap, err := snapshot.Parse(json.RawMessage(sampleUserInfo), time.Now().Add(-5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	fs := &fakeSnapshots{snap: snap, has: true}
	dir := t.TempDir()
	ms, _ := macros.NewStore(filepath.Join(dir, "macros.json"))
	rs, _ := rules.NewStore(filepath.Join(dir, "rules.json"))
	ev, _ := rules.NewEventLog(filepath.Join(dir, "events.jsonl"), 500)
	srv := httptest.NewServer((&Server{Auth: &fakeAuth{authorized: true}, Home: home, Macros: ms,
		Snapshots: fs, Rules: rs, Runner: &rules.Runner{Home: home, Macros: ms}, Events: ev, Engine: rules.NewEngine(nil)}).Handler())
	t.Cleanup(srv.Close)
	return srv, fs, rs
}

const ruleJSON = `{"name":"Кабинет жарко","when":[
  {"device_id":"sensor-office","property":"temperature","op":">","value":25},
  {"device_id":"blinds","capability":"open","op":">","value":50},
  {"time":{"after":"00:00","before":"23:59"}}],
 "then":[{"device_id":"blinds","type":"devices.capabilities.range","instance":"open","value":40}]}`

func TestHomeServedFromSnapshot(t *testing.T) {
	home := &fakeHome{userInfo: json.RawMessage(`{"status":"ok","devices":[]}`)}
	srv, _, _ := newRulesServer(t, home)
	req, _ := http.NewRequest("GET", srv.URL+"/api/home", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !strings.Contains(string(body), `"sensor-office"`) {
		t.Fatalf("home должен отдаваться из снимка: %d %s", resp.StatusCode, body[:60])
	}
	if age := resp.Header.Get("X-Snapshot-Age"); age == "" {
		t.Error("нет заголовка X-Snapshot-Age")
	}
	code, body2 := do(t, "GET", srv.URL+"/api/catalog", "")
	if code != 200 || !strings.Contains(body2, `"Кабинет"`) {
		t.Errorf("catalog из снимка: %d %s", code, body2[:80])
	}
}

func TestDeviceActionsKickPoller(t *testing.T) {
	home := &fakeHome{actionsResp: json.RawMessage(`{"status":"ok","devices":[]}`)}
	srv, fs, _ := newRulesServer(t, home)
	do(t, "POST", srv.URL+"/api/devices/actions", `{"devices":[{"id":"fan","actions":[{"type":"devices.capabilities.on_off","state":{"instance":"on","value":true}}]}]}`)
	if fs.kicks != 1 {
		t.Errorf("kicks = %d, want 1", fs.kicks)
	}
}

func TestRulesCRUDCheckRunEvents(t *testing.T) {
	home := &fakeHome{actionsResp: json.RawMessage(`{"status":"ok","request_id":"r","devices":[]}`)}
	srv, _, _ := newRulesServer(t, home)

	code, body := do(t, "POST", srv.URL+"/api/rules", ruleJSON)
	if code != 201 {
		t.Fatalf("create: %d %s", code, body)
	}
	var created rules.Rule
	decode(t, body, &created)
	if created.ID == "" || !created.Enabled || created.CooldownMinutes != 30 {
		t.Errorf("created = %+v", created)
	}

	code, body = do(t, "GET", srv.URL+"/api/rules/"+created.ID+"/check", "")
	var chk struct {
		OK         bool               `json:"ok"`
		Conditions []rules.CondResult `json:"conditions"`
	}
	decode(t, body, &chk)
	if code != 200 || !chk.OK || len(chk.Conditions) != 3 || chk.Conditions[0].Current != 26.3 {
		t.Errorf("check: %d %+v", code, chk)
	}

	code, body = do(t, "POST", srv.URL+"/api/rules/"+created.ID+"/run", "")
	if code != 200 || !strings.Contains(body, `"ok":true`) || home.gotActions == nil {
		t.Errorf("run: %d %s", code, body)
	}
	code, body = do(t, "GET", srv.URL+"/api/events?limit=10", "")
	var events []rules.Event
	decode(t, body, &events)
	if code != 200 || len(events) != 1 || events[0].RuleID != created.ID || !strings.Contains(events[0].Details, "вручную") {
		t.Errorf("events: %d %+v", code, events)
	}

	created.Enabled = false
	upd, _ := json.Marshal(created)
	if code, body = do(t, "PUT", srv.URL+"/api/rules/"+created.ID, string(upd)); code != 200 || !strings.Contains(body, `"enabled":false`) {
		t.Errorf("update: %d %s", code, body)
	}
	code, body = do(t, "GET", srv.URL+"/api/rules", "")
	if code != 200 || !strings.Contains(body, `"last_fired"`) {
		t.Errorf("list должен содержать last_fired после run: %d %s", code, body)
	}
	if code, _ = do(t, "DELETE", srv.URL+"/api/rules/"+created.ID, ""); code != 204 {
		t.Errorf("delete: %d", code)
	}
	if code, _ = do(t, "GET", srv.URL+"/api/rules/"+created.ID+"/check", ""); code != 404 {
		t.Errorf("check unknown: %d", code)
	}
}

func TestRulesValidationErrors(t *testing.T) {
	srv, _, _ := newRulesServer(t, &fakeHome{})
	code, body := do(t, "POST", srv.URL+"/api/rules", `{"name":"x","when":[{"device_id":"nope","property":"t","op":">","value":1}],"then":[{"device_id":"blinds","type":"devices.capabilities.range","instance":"open","value":40}]}`)
	if code != 400 || !strings.Contains(body, "устройство") {
		t.Errorf("validation: %d %s", code, body)
	}
	if code, _ := do(t, "POST", srv.URL+"/api/rules", `garbage`); code != 400 {
		t.Errorf("garbage: %d", code)
	}
}
```

Добавить в импорты теста `"io"` и `"net/http"`.

- [ ] **Step 3: Обработчики**

`internal/server/rule_handlers.go`:

```go
package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"smarthome/internal/rules"
)

type ruleView struct {
	rules.Rule
	LastFired *time.Time `json:"last_fired,omitempty"`
}

func (s *Server) ruleView(r rules.Rule) ruleView {
	v := ruleView{Rule: r}
	if s.Engine != nil {
		if t, ok := s.Engine.LastFired(r.ID); ok {
			v.LastFired = &t
		}
	}
	return v
}

func (s *Server) macroExists(id string) bool {
	if s.Macros == nil {
		return false
	}
	_, ok := s.Macros.Get(id)
	return ok
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	all := s.Rules.List()
	out := make([]ruleView, 0, len(all))
	for _, x := range all {
		out = append(out, s.ruleView(x))
	}
	writeJSON(w, http.StatusOK, out)
}

// decodeRule читает правило; enabled по умолчанию true.
func decodeRule(r *http.Request) (rules.Rule, error) {
	var body struct {
		rules.Rule
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return rules.Rule{}, errors.New("некорректный JSON правила")
	}
	rule := body.Rule
	rule.Enabled = body.Enabled == nil || *body.Enabled
	return rule, nil
}

func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	rule, err := decodeRule(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
		return
	}
	snap, _ := s.Snapshots.Latest()
	created, err := s.Rules.Create(rule, &snap, s.macroExists)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, s.ruleView(created))
}

func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	rule, err := decodeRule(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
		return
	}
	rule.ID = r.PathValue("id")
	snap, _ := s.Snapshots.Latest()
	if err := s.Rules.Update(rule, &snap, s.macroExists); err != nil {
		s.writeRuleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.ruleView(rule))
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	if err := s.Rules.Delete(r.PathValue("id")); err != nil {
		s.writeRuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) runRule(w http.ResponseWriter, r *http.Request) {
	rule, ok := s.Rules.Get(r.PathValue("id"))
	if !ok {
		s.writeRuleError(w, rules.ErrNotFound)
		return
	}
	details, err := s.Runner.Execute(r.Context(), rule)
	ev := rules.Event{Time: time.Now(), RuleID: rule.ID, RuleName: rule.Name, OK: err == nil, Details: "вручную: " + details}
	if err != nil {
		ev.Details += " — " + err.Error()
	}
	if s.Events != nil {
		_ = s.Events.Append(ev)
	}
	s.Snapshots.Kick()
	writeJSON(w, http.StatusOK, map[string]any{"ok": err == nil, "details": ev.Details})
}

func (s *Server) checkRule(w http.ResponseWriter, r *http.Request) {
	rule, ok := s.Rules.Get(r.PathValue("id"))
	if !ok {
		s.writeRuleError(w, rules.ErrNotFound)
		return
	}
	snap, has := s.Snapshots.Latest()
	if !has {
		var err error
		if snap, err = s.Snapshots.Refresh(r.Context()); err != nil {
			s.writeError(w, err)
			return
		}
	}
	okAll, conds := rules.Evaluate(rule, snap, time.Now())
	writeJSON(w, http.StatusOK, map[string]any{"ok": okAll, "conditions": conds,
		"snapshot_age": int(time.Since(snap.At).Seconds())})
}

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 500 {
		limit = 500
	}
	if s.Events == nil {
		writeJSON(w, http.StatusOK, []rules.Event{})
		return
	}
	writeJSON(w, http.StatusOK, s.Events.List(limit))
}

func (s *Server) writeRuleError(w http.ResponseWriter, err error) {
	if errors.Is(err, rules.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorBody{Error: "not_found", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusBadRequest, errorBody{Error: "bad_request", Message: err.Error()})
}
```

В `server.go`: добавить интерфейс `Snapshots` и поля; маршруты (регистрировать только если `s.Rules != nil`):

```go
	if s.Rules != nil {
		mux.HandleFunc("GET /api/rules", s.listRules)
		mux.HandleFunc("POST /api/rules", s.createRule)
		mux.HandleFunc("PUT /api/rules/{id}", s.updateRule)
		mux.HandleFunc("DELETE /api/rules/{id}", s.deleteRule)
		mux.HandleFunc("POST /api/rules/{id}/run", s.runRule)
		mux.HandleFunc("GET /api/rules/{id}/check", s.checkRule)
		mux.HandleFunc("GET /api/events", s.listEvents)
	}
```

В `home_handlers.go`: вспомогательная `func (s *Server) homeRaw(ctx) (json.RawMessage, time.Time, error)` — если `s.Snapshots == nil` → `s.Home.UserInfo(ctx)` и `time.Now()`; иначе `Latest()`, при `!has` → `Refresh(ctx)`. `home` и `catalog` используют её и ставят `w.Header().Set("X-Snapshot-Age", strconv.Itoa(int(time.Since(at).Seconds())))` до записи тела. В `deviceActions`, `runScenario` и в `macro_handlers.go` `runMacro` после успешного ответа — `if s.Snapshots != nil { s.Snapshots.Kick() }`. Существующие тесты (`TestHomeProxiesUserInfo` и др.) остаются валидными: у них `Snapshots == nil`.

- [ ] **Step 4: Точка входа**

`main.go` — после создания `client`, `macroStore`:

```go
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
```

и передать в `server.Server`: `Snapshots: pl, Rules: ruleStore, Runner: runner, Events: events, Engine: engine`. `http.ListenAndServe` заменить на `srv := &http.Server{Addr: addr, Handler: handler}`; `go func() { <-ctx.Done(); srv.Shutdown(context.Background()) }()`; `err := srv.ListenAndServe(); if errors.Is(err, http.ErrServerClosed) { return nil }`.

- [ ] **Step 5: Проверить**

Run: `gofmt -l . && go vet ./... && go test ./...` — зелёные. Дымовой запуск: `go build -o /tmp/sh-smoke . && YANDEX_CLIENT_ID=x YANDEX_CLIENT_SECRET=y PORT=18080 DATA_DIR=/tmp/smart-house-smoke /tmp/sh-smoke & PID=$!; sleep 1; curl -s -w '\n%{http_code}\n' localhost:18080/api/rules; curl -s -w '\n%{http_code}\n' localhost:18080/api/events; kill $PID`.
Expected: `[]` 200 и `[]` 200 (правила и события доступны без токена; `/api/home` без токена — 401, как раньше; в логе — предупреждения опроса «нет токена» раз в 10 с).

- [ ] **Step 6: Коммит**

```bash
git add internal/config internal/server main.go .env.example
git commit -m "feat: API автоматизаций, снимок дома в обработчиках, цикл правил"
```

### Task 7: Фронтенд — вкладка «Автоматизации»

**Files:**
- Modify: `web/src/api.js` (+ `rules`, `createRule`, `updateRule`, `deleteRule`, `runRule`, `checkRule`, `events`)
- Create: `web/src/describeRule.js`
- Create: `web/src/Automations.jsx`
- Modify: `web/src/Home.jsx` (третья вкладка)
- Modify: `web/src/styles.css`

**Interfaces:**
- Consumes: API Task 6; `home.devices` (имена устройств), `labels.js` (`label`, `unit`), `icons.jsx` (`Icon`), классы дизайна «Плитки» (`.tile`, `.sw`, `.badge`, `.btn`, `.icon-btn`, `.banner`, `.muted`).
- Produces: `describeRule(rule, deviceName) → { when: string[], then: string[] }`; компонент `<Automations home onUnauthorized />`.

- [ ] **Step 1: API-обёртки**

В `web/src/api.js` добавить:

```js
  rules: () => request('/api/rules'),
  createRule: (r) => request('/api/rules', json('POST', r)),
  updateRule: (r) => request(`/api/rules/${encodeURIComponent(r.id)}`, json('PUT', r)),
  deleteRule: (id) => request(`/api/rules/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  runRule: (id) => request(`/api/rules/${encodeURIComponent(id)}/run`, { method: 'POST' }),
  checkRule: (id) => request(`/api/rules/${encodeURIComponent(id)}/check`),
  events: (limit = 50) => request(`/api/events?limit=${limit}`),
```

- [ ] **Step 2: Человекочитаемое описание правила**

`web/src/describeRule.js`:

```js
import { label, unit } from './labels'

const OPS = { '<': '<', '<=': '≤', '>': '>', '>=': '≥', '==': '=', '!=': '≠' }

function fmtValue(v, u) {
  if (typeof v === 'boolean') return v ? 'вкл' : 'выкл'
  if (typeof v === 'number') return `${Math.round(v * 10) / 10}${u ? ' ' + u : ''}`
  if (v && typeof v === 'object') return 'цвет'
  return String(v)
}

// describeRule превращает условия и действия в короткие русские фразы.
// deviceName(id) → имя устройства; unitOf(id, instance) → единица (может быть пустой).
export function describeRule(rule, deviceName, unitOf = () => '') {
  const when = (rule.when || []).map((c) => {
    if (c.time) return `время ${c.time.after}–${c.time.before}`
    const inst = c.property || c.capability
    const u = unit(unitOf(c.device_id, inst))
    return `${deviceName(c.device_id)}: ${label(inst).toLowerCase()} ${OPS[c.op] || c.op} ${fmtValue(c.value, u)}`
  })
  const then = (rule.then || []).map((a) => {
    if (a.macro_id) return `макрос ${a.macro_id}`
    if (a.scenario_id) return `сценарий Яндекса ${a.scenario_id}`
    const u = unit(unitOf(a.device_id, a.instance))
    return `${deviceName(a.device_id)} → ${label(a.instance).toLowerCase()} ${fmtValue(a.value, u)}`
  })
  const extra = []
  if (rule.for_minutes) extra.push(`держится ${rule.for_minutes} мин`)
  if (rule.cooldown_minutes) extra.push(`не чаще чем раз в ${rule.cooldown_minutes} мин`)
  return { when, then, extra }
}
```

- [ ] **Step 3: Компонент**

`web/src/Automations.jsx`:

```jsx
import { useCallback, useEffect, useState } from 'react'
import { api } from './api'
import { Icon } from './icons'
import { describeRule } from './describeRule'

function fmtTime(iso) {
  if (!iso) return 'ещё не срабатывало'
  const d = new Date(iso)
  return d.toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

export default function Automations({ home, onUnauthorized }) {
  const [rules, setRules] = useState([])
  const [events, setEvents] = useState([])
  const [checks, setChecks] = useState({}) // id → результат /check
  const [error, setError] = useState('')

  const deviceName = useCallback(
    (id) => home?.devices?.find((d) => d.id === id)?.name || id,
    [home],
  )
  const unitOf = useCallback(
    (id, inst) => {
      const d = home?.devices?.find((x) => x.id === id)
      const p = d?.properties?.find((x) => x.parameters?.instance === inst)
      const c = d?.capabilities?.find((x) => (x.parameters?.instance || 'on') === inst)
      return p?.parameters?.unit || c?.parameters?.unit || ''
    },
    [home],
  )

  const load = useCallback(() => {
    Promise.all([api.rules(), api.events(50)])
      .then(([r, e]) => {
        setRules(r)
        setEvents(e)
        setError('')
      })
      .catch((e) => {
        if (e.status === 401) onUnauthorized()
        else setError(e.message)
      })
  }, [onUnauthorized])

  useEffect(() => {
    load()
  }, [load, home])

  const toggle = (rule) =>
    api.updateRule({ ...rule, enabled: !rule.enabled }).then(load).catch((e) => setError(e.message))
  const remove = (rule) => api.deleteRule(rule.id).then(load).catch((e) => setError(e.message))
  const run = (rule) =>
    api.runRule(rule.id).then(load).catch((e) => setError(e.message))
  const check = (rule) =>
    api
      .checkRule(rule.id)
      .then((res) => setChecks((c) => ({ ...c, [rule.id]: res })))
      .catch((e) => setError(e.message))

  return (
    <>
      <section className="room">
        <div className="room-head">
          <h2 className="display">Автоматизации</h2>
          <span className="muted">правила проверяются каждые 10 секунд</span>
        </div>
        {error && <div className="banner">{error}</div>}
        {!rules.length && (
          <div className="tile hint">
            <div className="name">Правил пока нет</div>
            <div className="state">
              Опишите Claude в терминале: «когда в кабинете больше 25 °C днём — закрывай жалюзи до 40 %» — правило появится здесь.
            </div>
          </div>
        )}
        <div className="rules">
          {rules.map((rule) => {
            const d = describeRule(rule, deviceName, unitOf)
            const chk = checks[rule.id]
            return (
              <div className={`tile rule${rule.enabled ? '' : ' off'}`} key={rule.id}>
                <div className="top">
                  <div>
                    <div className="name">{rule.name}</div>
                    <div className="state">последний раз: {fmtTime(rule.last_fired)}</div>
                  </div>
                  <label className="sw-wrap">
                    <input type="checkbox" checked={rule.enabled} onChange={() => toggle(rule)} aria-label="Включено" />
                    <span className={`sw${rule.enabled ? ' on' : ''}`}></span>
                  </label>
                </div>
                <div className="rule-body">
                  <div className="rule-col">
                    <div className="muted small">Если</div>
                    {d.when.map((w, i) => (
                      <div className="rule-line" key={i}>
                        {chk && <span className={`dot${chk.conditions?.[i]?.ok ? ' ok' : ' no'}`}></span>}
                        <span>{w}</span>
                        {chk && chk.conditions?.[i]?.current !== undefined && chk.conditions[i].kind !== 'time' && (
                          <span className="muted"> · сейчас {String(chk.conditions[i].current)}</span>
                        )}
                      </div>
                    ))}
                    {d.extra.length > 0 && <div className="muted small">{d.extra.join(' · ')}</div>}
                  </div>
                  <div className="rule-col">
                    <div className="muted small">То</div>
                    {d.then.map((t, i) => (
                      <div className="rule-line" key={i}>{t}</div>
                    ))}
                  </div>
                </div>
                <div className="row rule-actions">
                  <button className="btn" onClick={() => check(rule)}>
                    <Icon name="CHECK" size={16} /> Проверить
                  </button>
                  <button className="btn" onClick={() => run(rule)}>
                    <Icon name="PLAY" size={16} /> Выполнить сейчас
                  </button>
                  <button className="icon-btn danger" onClick={() => remove(rule)} aria-label="Удалить правило">
                    <Icon name="TRASH" size={16} />
                  </button>
                  {chk && (
                    <span className={`badge ${chk.ok ? 'ok' : 'busy'}`}>{chk.ok ? 'условия выполнены' : 'условия не выполнены'}</span>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </section>

      <section className="room">
        <div className="room-head">
          <h2 className="display">Журнал</h2>
          <span className="muted">последние {events.length}</span>
        </div>
        {!events.length ? (
          <p className="muted">Срабатываний ещё не было.</p>
        ) : (
          <div className="events">
            {events.map((e, i) => (
              <div className="event" key={i}>
                <span className="muted mono">{fmtTime(e.time)}</span>
                <span className={`badge ${e.ok ? 'ok' : 'err'}`}>{e.ok ? 'ок' : 'ошибка'}</span>
                <span className="event-name">{e.rule_name}</span>
                <span className="muted">{e.details}</span>
              </div>
            ))}
          </div>
        )}
      </section>
    </>
  )
}
```

Если в проекте компонент-тумблер реализован иначе (см. `Controls.jsx` — `Switch`), использовать его разметку/классы вместо `sw-wrap` — важно совпадение с уже принятым дизайном.

- [ ] **Step 4: Вкладка и стили**

`web/src/Home.jsx`: третья вкладка `automations` → «Автоматизации», рендер `<Automations home={home} onUnauthorized={onUnauthorized} />`. В шапке кнопка «Выключить свет» показывается только на вкладке устройств.

`web/src/styles.css` — добавить:

```css
.rules { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.tile.rule.off { opacity: .6; }
.rule-body { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.rule-col { display: flex; flex-direction: column; gap: 6px; font-size: 13px; }
.rule-line { display: flex; align-items: baseline; gap: 6px; }
.small { font-size: 12px; letter-spacing: .06em; text-transform: uppercase; }
.dot { width: 8px; height: 8px; border-radius: 4px; background: var(--muted); flex-shrink: 0; align-self: center; }
.dot.ok { background: var(--ok); } .dot.no { background: var(--danger); }
.rule-actions { flex-wrap: wrap; gap: 8px; }
.events { display: flex; flex-direction: column; gap: 6px; }
.event { display: grid; grid-template-columns: 110px auto 1fr; gap: 10px; align-items: center; padding: 10px 14px; background: var(--surface); border-radius: 14px; font-size: 13px; }
.event .event-name { font-weight: 600; }
.event .muted:last-child { grid-column: 2 / -1; }
.mono { font-variant-numeric: tabular-nums; }
@media (max-width: 700px) { .rules { grid-template-columns: 1fr; } .rule-body { grid-template-columns: 1fr; } .event { grid-template-columns: 1fr; } }
```

- [ ] **Step 5: Собрать и проверить**

`cd web && npm run build` без предупреждений; `go build -o /dev/null .`. Дымовой запуск на 18080 с фейковыми ключами: `curl localhost:18080/` отдаёт `index.html`; `/api/rules` → `[]`.

- [ ] **Step 6: Коммит**

```bash
git add web/src
git commit -m "feat: вкладка «Автоматизации» — правила, проверка условий, журнал"
```

---

### Task 8: Skill и README — автоматизации и понимание ситуации

**Files:**
- Modify: `.claude/skills/smart-home/SKILL.md`
- Modify: `README.md`

- [ ] **Step 1: Раздел «Понимать ситуацию, а не только команду»** (вставить после раздела 2 «Узнать, какие устройства есть»):

````markdown
## 2а. Понимать ситуацию, а не только команду

Владелец часто описывает ситуацию, а не устройство: «в кабинете темно», «солнце в глаза», «душно в спальне», «холодно в гостиной». Действовать по смыслу:

- **Комната** — из фразы; устройства этой комнаты — из каталога (`room`).
- **Темно / включи свет** → основной свет комнаты: устройство типа `light.ceiling` или с «Большой свет» в имени; настольные лампы, ночник, ленты и подсветки — только если сказано про рабочее место, ночь или подсветку.
- **Солнце в глаза / ярко / жарко от солнца** → шторы или жалюзи комнаты (`openable.curtain`, умение `open`): прикрыть до 30–40 %, а не закрывать наглухо, если не просили.
- **Душно / жарко** → вентилятор (`ventilation.fan`) или кондиционер; ночью (22:00–08:00) — низкая скорость. **Сухо** → увлажнитель. **Пыльно / плохой воздух** → очиститель.
- **Холодно** → термостат/батареи комнаты: поднять `temperature` на 1–2 °C от текущего, не выше 25.
- **Спать / ночь** → выключить основной свет и ТВ комнаты, ночник — на минимум.
- Минимальное вмешательство: одно-два устройства, ближайшие к смыслу фразы; остальное не трогать.
- Неоднозначно (три лампочки на кухне, два телевизора) — спросить одним вопросом, предложив варианты из каталога.
- Никогда не трогать без явной просьбы: краны/клапаны воды (`Нептун`), сигнализацию, кормушку, чайник (нагрев), розетки бытовой техники.
- После действия — короткий отчёт: что и на сколько изменено, что не сработало. Если ситуация повторяется («каждый день солнце в глаза») — предложить автоматизацию (раздел 5).
````

- [ ] **Step 2: Раздел «5. Автоматизации (правила)»** (после раздела «4. Выполнить»):

````markdown
## 5. Автоматизации (правила)

Правило = условия по «И» + действия; сервер проверяет правила на каждом опросе (раз в 10 с) и выполняет действия сам. Реакция — до 10 секунд, поэтому мгновенное «свет по движению» лучше оставить сценариям Яндекса; правила — для освещённости, температуры, влажности, PM2.5, мощности, времени и состояния устройств.

```json
{
  "name": "Кабинет жарко → жалюзи",
  "when": [
    {"device_id": "<датчик климата кабинета>", "property": "temperature", "op": ">", "value": 25},
    {"device_id": "<жалюзи>", "capability": "open", "op": ">", "value": 50},
    {"time": {"after": "11:00", "before": "17:00"}}
  ],
  "for_minutes": 0,
  "cooldown_minutes": 60,
  "then": [{"device_id": "<жалюзи>", "type": "devices.capabilities.range", "instance": "open", "value": 40}]
}
```

- `property` — показание датчика (`instance` из `properties` каталога), `capability` — состояние умения (`instance` из `capabilities`), `time` — окно по местному времени сервера (`after > before` — через полночь). Операторы `< <= > >= == !=`.
- `for_minutes` — условия должны держаться N минут (для «воздух чистый 20 минут»); `cooldown_minutes` — не повторять раньше (по умолчанию 30). Срабатывает, когда условия стали истинными; повтор — только после того, как они стали ложными и снова истинными.
- Действия — как у макросов, плюс `{"macro_id": "…"}` и `{"scenario_id": "…"}`. «Или» — два правила с одинаковыми действиями.
- Порог неизвестен (например, освещённость на теневом балконе) — сначала посмотреть текущее значение в каталоге в разное время дня, поставить порог с запасом и сказать владельцу, что его можно подвинуть.

```bash
curl -s -X POST $BASE/api/rules -H 'Content-Type: application/json' -d @rule.json   # создать (201)
curl -s $BASE/api/rules                                                             # список с last_fired
curl -s -X PUT $BASE/api/rules/<id> -H 'Content-Type: application/json' -d @rule.json  # изменить / включить-выключить (enabled)
curl -s -X DELETE $BASE/api/rules/<id>
curl -s $BASE/api/rules/<id>/check    # почему (не) сработало: по каждому условию ok и текущее значение
curl -s -X POST $BASE/api/rules/<id>/run   # выполнить действия сейчас, минуя условия
curl -s "$BASE/api/events?limit=50"   # журнал срабатываний
```

Перед сохранением показать владельцу таблицу «условие → действие» с именами устройств и значениями; после — вызвать `/check` и сказать, выполнены ли условия сейчас. Ошибка 400 содержит причину (устройство не найдено, режим вне списка, значение вне диапазона).
````

- [ ] **Step 3: README** — раздел «Автоматизации» перед «Разработка»: что это, ограничение 10 секунд, где хранятся (`data/rules.json`, `data/events.jsonl`), `RULES_ENABLED=false` для второй копии сервера (на ноутбуке разработчика), ссылка на spec.

- [ ] **Step 4: Коммит**

```bash
git add .claude/skills/smart-home/SKILL.md README.md
git commit -m "docs: skill — понимание ситуации и автоматизации; README"
```

---

### Task 9: Приёмка — стартовые правила на домашнем сервере

Выполняет контролёр с владельцем; субагенту не делегировать.

- [ ] **Step 1: Деплой.** `make deploy DEPLOY=user@192.168.1.95`; на ноутбуке в `.env` выставить `RULES_ENABLED=false` и перезапустить локальный сервер (или остановить его). Проверить `curl $BASE/api/rules` → `[]`.
- [ ] **Step 2: Создать 13 правил** из спеки через skill (реальные `device_id` из `/api/catalog` домашнего сервера), показать владельцу таблицу, для каждого вызвать `/check`.
- [ ] **Step 3: Проверить движок вживую.** Временное правило с заведомо истинным условием (например, `Вентилятор on == false` без окна времени → включить Ночник на 1 %) — убедиться, что в течение 10–20 с оно сработало, запись появилась в журнале и во вкладке; удалить правило и вернуть ночник.
- [ ] **Step 4: Порог освещённости.** Записать значения `illumination` балконного датчика днём и вечером (`/api/catalog`), при необходимости подвинуть пороги «стемнело» (30 лк) и «пасмурно» (200 лк).
- [ ] **Step 5: UI.** Вкладка «Автоматизации»: список, переключатель, «Проверить» с текущими значениями, журнал; тёмная тема; телефон.

---

## Самопроверка плана

**Покрытие спеки:** стартовый набор (Task 9); модель правила, валидация, «И», `for_minutes`, кулдаун, фронт, окно через полночь (Tasks 3–4); действия одним запросом + макрос + сценарий (Task 5); опрос на сервере, `/api/home` и `/api/catalog` из снимка, `X-Snapshot-Age`, `Kick` после действий (Tasks 2, 6); движок на каждом снимке, журнал `events.jsonl`, 500 в памяти (Tasks 5–6); API (Task 6); вкладка (Task 7); skill (Task 8); тесты Go (Tasks 1–6). Понимание ситуации в skill — по просьбе владельца (Task 8, раздел 2а).

**Согласованность имён:** `snapshot.Snapshot.Value` используется движком; `poller.Poller` реализует `server.Snapshots` (`Latest`, `Refresh`, `Kick`); `rules.Home` реализует `*yandex.Client`; `rules.Runner.Execute` возвращает `(details, err)` и в `main.go`, и в обработчике `run`; `CondResult` сериализуется полями `index/ok/current/kind`, которые читает `Automations.jsx`; `last_fired` — только в `ruleView` (Engine.LastFired), в файле правил не хранится.

**Известные упрощения:** состояние срабатываний живёт в памяти (после перезапуска сервера правило может сработать повторно, если условия истинны — защищает кулдаун от частого повтора не сразу после старта; приемлемо); `macro_id` в описании правила показывается как id, а не имя (UI может подставить имя из `/api/macros` — при желании); события не ротируются (файл только растёт, ~100 байт на событие).
