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

	"smarthome/internal/snapshot"
)

const sampleUserInfo = `{"status":"ok","request_id":"r",
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
 "scenarios":[{"id":"s1","name":"Я дома","is_active":true}]}`

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
