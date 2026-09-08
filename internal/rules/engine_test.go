package rules

import (
	"encoding/json"
	"strings"
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

// TestEvaluateObjectValuedCurrentDoesNotPanic проверяет, что сравнение с
// объектным/массивным значением умения (как color_setting в режиме hsv у
// Яндекса) не паникует на ==/!=, а просто не проходит условие.
func TestEvaluateObjectValuedCurrentDoesNotPanic(t *testing.T) {
	s := snapWith(t, 26.3, 100, at(12, 0))
	hsv := map[string]any{"h": 200.0, "s": 50.0, "v": 100.0}
	s.Devices["blinds"].Caps["open"] = hsv // подменяем значение умения на объект

	rule := Rule{When: []Condition{{DeviceID: "blinds", Capability: "open", Op: "==", Value: hsv}}}
	ok, res := Evaluate(rule, s, at(12, 0))
	if ok || res[0].OK {
		t.Errorf("сравнение объектов должно быть false без паники: ok=%v res=%+v", ok, res)
	}

	list := []any{"a", "b"}
	s.Devices["blinds"].Caps["open"] = list // и массив тоже не должен вызывать панику
	rule2 := Rule{When: []Condition{{DeviceID: "blinds", Capability: "open", Op: "!=", Value: list}}}
	if ok, res := Evaluate(rule2, s, at(12, 0)); ok || res[0].OK {
		t.Errorf("сравнение массивов должно быть false без паники: ok=%v res=%+v", ok, res)
	}
}

// TestCondResultMarshalsFalseCurrent фиксирует, что нулевые значения
// (false, 0, "") в Current сериализуются в JSON, а не пропадают из-за
// omitempty — самое частое значение для выключенного умения.
func TestCondResultMarshalsFalseCurrent(t *testing.T) {
	b, err := json.Marshal(CondResult{Index: 0, OK: false, Current: false, Kind: "capability"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"current":false`) {
		t.Errorf(`ожидали "current":false в %s`, b)
	}
}

// TestTickDisabledRuleDoesNotFireAndResetsState — выключенное правило не
// срабатывает даже когда снимок «горячий» и условия уже выполнялись
// for_minutes подряд (без гварда — сработало бы). После повторного
// включения состояние сброшено: срабатывание требует нового отсчёта
// for_minutes от момента включения, а не продолжения старого.
func TestTickDisabledRuleDoesNotFireAndResetsState(t *testing.T) {
	now := at(12, 0)
	e := NewEngine(func() time.Time { return now })
	r := officeHot()
	r.ID = "r3"
	r.ForMinutes = 5
	r.CooldownMinutes = 0
	hot := func(min int) snapshot.Snapshot { return snapWith(t, 26.3, 100, at(12, min)) }

	// Взводим правило: условия истинны с 12:00.
	e.Tick([]Rule{r}, hot(0))

	// К 12:05 условия истинны уже for_minutes=5 минут — без гварда сработало бы.
	// Но правило выключено прямо к этому тику.
	now = at(12, 5)
	r.Enabled = false
	if fired := e.Tick([]Rule{r}, hot(5)); len(fired) != 0 {
		t.Fatalf("выключенное правило не должно срабатывать, даже если условия готовы: %+v", fired)
	}

	// Включаем обратно: снимок всё ещё горячий, но состояние сброшено.
	r.Enabled = true
	now = at(12, 6)
	if fired := e.Tick([]Rule{r}, hot(6)); len(fired) != 0 {
		t.Fatal("сразу после включения не должно срабатывать — нужен новый отсчёт for_minutes")
	}
	now = at(12, 10)
	if fired := e.Tick([]Rule{r}, hot(10)); len(fired) != 0 {
		t.Fatal("4 минуты с момента повторного включения < for_minutes=5")
	}
	now = at(12, 11)
	if fired := e.Tick([]Rule{r}, hot(11)); len(fired) != 1 {
		t.Fatalf("должно сработать через for_minutes после повторного включения: %+v", fired)
	}
}
