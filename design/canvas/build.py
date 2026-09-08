# Собирает артборды дизайна «Плитки» из общих стилей и иконок.
import re, pathlib
D = pathlib.Path(__file__).parent
LIGHT = (D / "_light.css").read_text()
ICONS = dict(l.split(":", 1) for l in (D / "_icons.txt").read_text().strip().splitlines())
FONTS = '<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Unbounded:wght@500;600&family=Onest:wght@400;500;600&display=swap">'
DARK = """
    body { background: #0f1318; color: #eef1f6; }
    .tab { background: #1a2029; color: #8b97a7; } .tab.active { background: #eef1f6; color: #0f1318; }
    .chip { background: #1a2029; color: #8b97a7; } .chip.active { background: #eef1f6; color: #0f1318; }
    .btn { background: #1a2029; color: #eef1f6; } .btn.primary { background: #eef1f6; color: #0f1318; }
    .icon-btn { background: #1a2029; color: #8b97a7; }
    .tile { background: #1a2029; color: #eef1f6; } .tile .state { color: #8b97a7; } .tile .ico { background: rgba(255,255,255,.06); }
    .tile[class*="on-"] { color: #16202b; } .tile[class*="on-"] .state { color: rgba(22,32,43,.62); } .tile[class*="on-"] .ico { background: rgba(22,32,43,.06); }
    .sw { background: rgba(255,255,255,.14); } .sw::after { background: #eef1f6; } .sw.on { background: #eef1f6; } .sw.on::after { background: #0f1318; }
    .tile[class*="on-"] .sw.on { background: #16202b; } .tile[class*="on-"] .sw.on::after { background: #ffffff; }
    .slider .lbl { color: #8b97a7; } .slider .track { background: rgba(255,255,255,.14); } .slider .fill { background: #eef1f6; } .slider .knob { background: #0f1318; border-color: #eef1f6; }
    .tile[class*="on-"] .slider .lbl { color: rgba(22,32,43,.62); } .tile[class*="on-"] .slider .track { background: rgba(22,32,43,.12); } .tile[class*="on-"] .slider .fill { background: #16202b; } .tile[class*="on-"] .slider .knob { background: #ffffff; border-color: #16202b; }
    .sel { background: rgba(255,255,255,.08); } .tile[class*="on-"] .sel { background: rgba(22,32,43,.06); }
    .badge.busy { background: rgba(255,255,255,.08); color: #8b97a7; }
    .input { background: #1a2029; color: #eef1f6; } .input.focus { border-color: #eef1f6; }
    .muted, .unit { color: #8b97a7; }
    .swatch { box-shadow: 0 0 0 1px rgba(255,255,255,.18); } .swatch.sel { box-shadow: 0 0 0 2px #eef1f6; }
    .tile .state.err { color: #ffb4ab; } .badge.ok { background: rgba(31,122,85,.35); color: #9fe3d2; } .badge.err { background: rgba(179,38,30,.35); color: #ffb4ab; }
    .banner { background: rgba(179,38,30,.3); color: #ffb4ab; }
"""

def svg(name, size=20, sw=1.8):
    return (f'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="{sw}" stroke-linecap="round" '
            f'stroke-linejoin="round" style="width: {size}px; height: {size}px;">{ICONS[name]}</svg>')

CHEV = lambda: svg("CHEV", 14, 2)
OK = lambda: 'Выполнено ' + svg("CHECK", 12, 2.4)

def doc(body, dark=False, extra=""):
    return ("<!doctype html>\n<html>\n<head>\n  <meta charset=\"utf-8\">\n  <script src=\"./support.js\"></script>\n</head>\n<body>\n<x-dc>\n<helmet>\n  "
            + FONTS + "\n  <style>\n" + LIGHT + (DARK if dark else "") + extra + "  </style>\n</helmet>\n" + body + "\n</x-dc>\n</body>\n</html>\n")

def header(active="devices", right=True):
    tabs = (f'<span class="tab {"active" if active=="devices" else ""}">Устройства</span>'
            f'<span class="tab {"active" if active=="scenarios" else ""}">Сценарии</span>')
    r = ('<div style="display: flex; gap: 8px;"><span class="btn">Выключить всё</span>'
         f'<span class="icon-btn">{svg("REFRESH",18)}</span></div>') if right else ''
    return ('<header style="display: flex; align-items: center; justify-content: space-between;">'
            '<div class="display" style="font-size: 20px; font-weight: 600;">Умный дом</div>'
            f'<div style="display: flex; gap: 8px;">{tabs}</div>{r}</header>')

ROOMS = [("Спальня",14),("Кабинет",9),("Кухня",8),("Коридор",8),("Гостиная",6),("Ванная",3),("Постирочная",3),("Балкон",2)]
def chips(active="Спальня"):
    items = "".join(f'<span class="chip {"active" if n==active else ""}">{n} <span style="opacity: .6;">{c}</span></span>' for n,c in ROOMS)
    return f'<div style="display: flex; gap: 8px; overflow: hidden;">{items}<span class="chip">Все 54</span></div>'

def slider(lbl, pct, val):
    return (f'<div class="slider"><span class="lbl">{lbl}</span><div class="track"><div class="fill" style="width: {pct}%;"></div>'
            f'<div class="knob" style="left: {pct}%;"></div></div><span class="val">{val}</span></div>')

def swatches(sel=0):
    cols = ["#ffd27a","#ff9a6c","#8fb7e8","#9fe3d2","#e9b7ff","#ffffff"]
    return '<div class="swatches">' + "".join(f'<span class="swatch {"sel" if i==sel else ""}" style="background: {c};"></span>' for i,c in enumerate(cols)) + '</div>'

def tile(icon, name, state, cls="", sw=None, body="", state_cls=""):
    top_r = f'<div class="sw {"on" if sw else ""}"></div>' if sw is not None else ""
    ico = f'<div class="ico">{svg(icon)}</div>' if icon else ""
    return (f'<div class="tile {cls}"><div class="top">{ico}{top_r}</div>'
            f'<div style="display: flex; flex-direction: column; gap: 10px;"><div><div class="name">{name}</div>'
            f'<div class="state {state_cls}">{state}</div></div>{body}</div></div>')

def climate_tile(t, h, p, batt):
    return (f'<div class="tile"><div class="top"><div><div class="name">Климат</div><div class="state">датчик · батарея {batt}</div></div></div>'
            f'<div style="display: flex; gap: 18px; align-items: flex-end;"><div><span class="big">{t}</span><span class="unit">°C</span></div>'
            f'<div><span class="big">{h}</span><span class="unit">%</span></div><div><span class="big">{p}</span><span class="unit">мм рт. ст.</span></div></div></div>')

def section(title, sub, tiles, cols=4):
    return (f'<section style="display: flex; flex-direction: column; gap: 14px;"><div style="display: flex; align-items: baseline; gap: 14px;">'
            f'<h2 class="display" style="margin: 0; font-size: 26px; font-weight: 600;">{title}</h2><span class="muted" style="font-size: 13px;">{sub}</span></div>'
            f'<div style="display: grid; grid-template-columns: repeat({cols}, minmax(0, 1fr)); gap: 14px;">{"".join(tiles)}</div></section>')

def devices_body(dark):
    bg = "#0f1318" if dark else "#eef1f6"
    bedroom = [
        climate_tile("24,6","51","757","76 %"),
        tile("AIR","Очиститель воздуха","работает · PM2.5 3 мкг/м³ · 51 %","on-air",True,
             '<div class="row" style="justify-content: space-between;"><span class="state">Подсветка</span><div class="sw" style="transform: scale(.85); transform-origin: right center;"></div></div>'),
        tile("CURTAIN","Шторы","открыты на 20 %","on-curtain",True, slider("Открыть", 20, "20 %")),
        tile("LAMP","Левая лампочка","включена · 3400 K","on-light",True, slider("Яркость",100,"100 %") + swatches(0)),
        tile("LAMP","Правая лампочка","выключена · 3400 K","",False, slider("Яркость",100,"100 %")),
        tile("LAMP","Ночник","выключен · цвет","",False, slider("Яркость",1,"1 %")),
        tile("LAMP","Настольная лампа","выключена · 3950 K","",False, slider("Яркость",100,"100 %")),
        tile("LAMP","Большой свет","выключен","",False),
        tile("DROP","Увлажнитель",'<span class="badge busy">применяем…</span>',"busy",True,
             '<div class="sel"><span>Скорость</span><span class="row" style="gap: 4px;">турбо ' + CHEV() + '</span></div>'),
        tile("FAN","Вентилятор","выключен · вращение включено","",False,'<div class="sel"><span>Скорость</span><span class="row" style="gap: 4px;">низкая ' + CHEV() + '</span></div>'),
        tile("THERMO","Батареи","сейчас 23,3 °C","",None, slider("Задать", 30, "22 °C")),
        tile("PLUG","Фумигатор","Питание: устройство не отвечает","",False,"", "err"),
        tile("TV","Телевизор","выключен","",False),
        tile("CAM","Камера","движение · микрофон выключен","",None),
    ]
    office = [
        tile("LAMP","Настольная лампа","включена · 3900 K","on-light",True, slider("Яркость",90,"90 %")),
        tile("THERMO","Батареи","сейчас 25,9 °C · батарея 98 %","",None, slider("Задать",57,"22 °C")),
        tile("CURTAIN","Жалюзи","открыты на 100 %","on-curtain",True, slider("Открыть",100,"100 %")),
        tile("PLUG","Фитолампа","выключена · 233 В · 0 Вт","",False),
    ]
    return (f'<div style="width: 1280px; height: 1080px; overflow: hidden; background: {bg}; display: flex; flex-direction: column; gap: 22px; padding: 26px 32px; box-sizing: border-box;">'
            + header("devices") + chips("Спальня") + section("Спальня","14 устройств · 3 включено", bedroom)
            + section("Кабинет","9 устройств · 2 включено", office) + '</div>')

def login_body():
    steps = ('<div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px;">'
             '<div style="background: #eef1f6; border-radius: 16px; padding: 14px;"><div class="display" style="font-size: 18px;">1</div><div style="margin-top: 6px; font-size: 13px;"><a href="#">Открыть страницу Яндекса</a> и разрешить доступ</div></div>'
             '<div style="background: #eef1f6; border-radius: 16px; padding: 14px;"><div class="display" style="font-size: 18px;">2</div><div style="margin-top: 6px; font-size: 13px;">Скопировать код подтверждения и вставить сюда</div></div></div>')
    card = ('<div class="tile" style="width: 520px; gap: 22px; min-height: 0; padding: 28px;">'
            '<div><div class="display" style="font-size: 26px; font-weight: 600;">Умный дом</div><div class="muted" style="margin-top: 6px;">Вход через Яндекс — один раз, дальше помним</div></div>'
            + steps +
            '<div class="row"><div class="input focus" style="flex-grow: 1; background: #eef1f6;"><span class="ph">Код подтверждения</span></div><span class="btn primary disabled">Войти</span></div></div>')
    err = ('<div class="tile" style="width: 520px; gap: 22px; min-height: 0; padding: 28px;">'
           '<div><div class="display" style="font-size: 26px; font-weight: 600;">Умный дом</div><div class="muted" style="margin-top: 6px;">Вход через Яндекс — один раз, дальше помним</div></div>'
           + steps +
           '<div class="row"><div class="input" style="flex-grow: 1; background: #eef1f6; font-variant-numeric: tabular-nums;">1234567</div><span class="btn primary">Войти</span></div>'
           '<div class="banner">Не удалось войти: код истёк, запросите новый на странице Яндекса</div></div>')
    return ('<div style="width: 1280px; height: 720px; overflow: hidden; background: #eef1f6; box-sizing: border-box; padding: 26px 32px; display: flex; flex-direction: column; gap: 40px;">'
            '<div class="display" style="font-size: 20px; font-weight: 600;">Умный дом</div>'
            '<div style="display: flex; gap: 60px; align-items: flex-start; justify-content: center;">'
            f'<div style="display: flex; flex-direction: column; gap: 10px;"><span class="muted" style="font-size: 12px; letter-spacing: .08em; text-transform: uppercase;">Обычное состояние</span>{card}</div>'
            f'<div style="display: flex; flex-direction: column; gap: 10px;"><span class="muted" style="font-size: 12px; letter-spacing: .08em; text-transform: uppercase;">Ошибка</span>{err}</div></div></div>')

SCEN = ["Я дома","Время сна","Выключить весь свет","Вечерний свет в спальне","Открыть жалюзи","Закрыть жалюзи","Поставить на сигнализацию","Снять с сигнализации",
        "Включить подсветку на балконе","Выключить все на кухне","Вода перекрыта","Время еды"]
def scenario_tile(name, status=None):
    badge = {"ok": '<span class="badge ok">' + OK() + '</span>', "busy": '<span class="badge busy">Запускаем…</span>',
             "err": '<span class="badge err">не отвечает</span>'}.get(status, "")
    return (f'<div class="tile" style="min-height: 96px; gap: 10px;"><div class="top"><div class="name" style="font-size: 14px;">{name}</div>'
            f'<span class="icon-btn" style="width: 36px; height: 36px; border-radius: 12px; background: #eef1f6; color: #16202b;">{svg("PLAY",16)}</span></div>'
            f'<div>{badge}</div></div>')

def macro_tile(name, n, devices, status=None, cls=""):
    badge = {"ok": '<span class="badge ok">' + OK() + '</span>'}.get(status, "")
    return (f'<div class="tile {cls}" style="min-height: 0; gap: 16px;"><div class="top"><div><div class="name">{name}</div>'
            f'<div class="state">{n} · {devices}</div></div>{badge}</div>'
            f'<div class="row"><span class="btn primary">{svg("PLAY",16)} Запустить</span><span class="icon-btn" style="background: #eef1f6;">{svg("PEN",16)}</span>'
            f'<span class="icon-btn danger" style="background: #eef1f6; color: #e5484d;">{svg("TRASH",16)}</span></div></div>')

def scenarios_body():
    stiles = [scenario_tile(n, {"Я дома":"ok","Время сна":"busy","Вода перекрыта":"err"}.get(n)) for n in SCEN]
    ysec = ('<section style="display: flex; flex-direction: column; gap: 14px;"><div style="display: flex; align-items: baseline; gap: 14px;">'
            '<h2 class="display" style="margin: 0; font-size: 26px; font-weight: 600;">Сценарии Яндекса</h2><span class="muted" style="font-size: 13px;">25 · создаются в приложении «Дом с Алисой»</span>'
            '<span class="chip" style="margin-left: auto;">Показать ещё 13</span></div>'
            f'<div style="display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px;">{"".join(stiles)}</div></section>')
    empty = ('<div class="tile" style="min-height: 0; border: 2px dashed rgba(22,32,43,.15); background: transparent; justify-content: center; gap: 8px;">'
             '<div class="name">Макросы собирает Claude</div><div class="state">Напишите в терминале: «сделай сценарий кино — выключи большой свет, торшер на 30 %» — макрос появится здесь.</div></div>')
    msec = ('<section style="display: flex; flex-direction: column; gap: 14px;"><div style="display: flex; align-items: baseline; gap: 14px;">'
            '<h2 class="display" style="margin: 0; font-size: 26px; font-weight: 600;">Мои макросы</h2><span class="muted" style="font-size: 13px;">хранятся локально</span>'
            f'<span class="btn" style="margin-left: auto;">{svg("PLUS",16)} Новый макрос</span></div>'
            '<div style="display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px;">'
            + macro_tile("Кино","4 действия","Большой свет, Торшер, Телевизор","ok")
            + macro_tile("Ухожу","6 действий","весь свет, Шторы, Очиститель")
            + empty + '</div></section>')
    return ('<div style="width: 1280px; height: 960px; overflow: hidden; background: #eef1f6; display: flex; flex-direction: column; gap: 22px; padding: 26px 32px; box-sizing: border-box;">'
            + header("scenarios", right=False) + ysec + msec + '</div>')

def action_row(dev, cap, val, kind="sel"):
    v = {"sel": f'<div class="sel" style="background: #eef1f6; justify-content: space-between;"><span>{val}</span>' + CHEV() + '</div>',
         "num": f'<div class="input" style="background: #eef1f6; padding: 8px 12px; border-radius: 12px; font-weight: 500; font-variant-numeric: tabular-nums; justify-content: space-between;"><span>{val}</span><span class="muted">%</span></div>',
         "color": f'<div class="row" style="padding: 6px 12px; background: #eef1f6; border-radius: 12px;"><span class="swatch sel" style="background: {val};"></span><span class="muted" style="font-size: 13px;">{val}</span></div>'}[kind]
    return ('<div style="display: grid; grid-template-columns: 1.3fr 1fr 1fr 42px; gap: 10px; align-items: center;">'
            f'<div class="sel" style="background: #eef1f6; justify-content: space-between;"><span>{dev}</span>' + CHEV() + '</div>'
            f'<div class="sel" style="background: #eef1f6; justify-content: space-between;"><span>{cap}</span>' + CHEV() + '</div>{v}'
            f'<span class="icon-btn" style="background: #eef1f6; color: #5b6a7c;">{svg("X",16)}</span></div>')

def editor_body():
    unavailable = ('<div style="display: grid; grid-template-columns: 1fr 42px; gap: 10px; align-items: center;">'
                   '<div class="banner" style="padding: 10px 14px;">Устройство недоступно (id 4f2c…) — убрано из дома в приложении Яндекса</div>'
                   f'<span class="icon-btn" style="background: #eef1f6; color: #5b6a7c;">{svg("X",16)}</span></div>')
    card = ('<div class="tile" style="width: 880px; min-height: 0; gap: 20px; padding: 28px;">'
            '<div class="top"><div><div class="display" style="font-size: 22px; font-weight: 600;">Новый макрос</div><div class="state" style="margin-top: 4px;">Выполняется одним запросом к Яндексу</div></div>'
            '<span class="badge busy">не сохранён</span></div>'
            '<div style="display: flex; flex-direction: column; gap: 8px;"><span class="muted" style="font-size: 13px;">Название</span><div class="input focus" style="background: #eef1f6; font-weight: 500;">Кино</div></div>'
            '<div style="display: flex; flex-direction: column; gap: 10px;"><span class="muted" style="font-size: 13px;">Действия</span>'
            + action_row("Большой свет в гостиной","Питание","выключить")
            + action_row("Торшер","Яркость","30","num")
            + action_row("Торшер","Цвет","#ffd27a","color")
            + action_row("Телевизор в гостиной","Питание","включить")
            + unavailable + '</div>'
            f'<div class="row" style="justify-content: space-between;"><span class="btn" style="background: #eef1f6;">{svg("PLUS",16)} Действие</span>'
            '<div class="row"><span class="btn" style="background: transparent; color: #5b6a7c;">Отмена</span><span class="btn primary">Сохранить</span></div></div></div>')
    return ('<div style="width: 1280px; height: 760px; overflow: hidden; background: #eef1f6; display: flex; flex-direction: column; gap: 22px; padding: 26px 32px; box-sizing: border-box;">'
            + header("scenarios", right=False) + f'<div style="display: flex; justify-content: center;">{card}</div></div>')

def mobile_body():
    tiles = [
        climate_tile("24,6","51","757","76 %").replace('<div class="tile">','<div class="tile" style="grid-column: span 2; min-height: 120px;">',1),
        tile("AIR","Очиститель","работает · PM2.5 3 мкг/м³","on-air",True),
        tile("CURTAIN","Шторы","открыты на 20 %","on-curtain",True, slider("",20,"20 %").replace('<span class="lbl"></span>','')),
        tile("LAMP","Левая лампочка","включена","on-light",True, slider("",100,"100 %").replace('<span class="lbl"></span>','')),
        tile("LAMP","Правая лампочка","выключена","",False),
    ]
    chips_m = "".join(f'<span class="chip {"active" if n=="Спальня" else ""}">{n}</span>' for n,_ in ROOMS[:4])
    return ('<div style="width: 390px; height: 844px; overflow: hidden; background: #eef1f6; display: flex; flex-direction: column; gap: 16px; padding: 18px 16px; box-sizing: border-box;">'
            '<header style="display: flex; align-items: center; justify-content: space-between;"><div class="display" style="font-size: 18px; font-weight: 600;">Умный дом</div>'
            f'<span class="icon-btn">{svg("REFRESH",18)}</span></header>'
            '<div style="display: flex; gap: 6px; background: #ffffff; padding: 4px; border-radius: 16px;"><span class="tab active" style="flex: 1; text-align: center; padding: 9px;">Устройства</span><span class="tab" style="flex: 1; text-align: center; padding: 9px;">Сценарии</span></div>'
            f'<div style="display: flex; gap: 8px; overflow: hidden;">{chips_m}</div>'
            '<div style="display: flex; align-items: baseline; gap: 10px;"><h2 class="display" style="margin: 0; font-size: 22px; font-weight: 600;">Спальня</h2><span class="muted" style="font-size: 12px;">3 из 14 включено</span></div>'
            f'<div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px;">{"".join(tiles)}</div></div>')

def components_body():
    def sw_block(hexv, name, dark_text=False):
        return (f'<div style="display: flex; flex-direction: column; gap: 6px;"><div style="height: 56px; border-radius: 14px; background: {hexv}; box-shadow: inset 0 0 0 1px rgba(22,32,43,.08);"></div>'
                f'<div style="font-size: 12px;"><b>{name}</b><br><span class="muted" style="font-variant-numeric: tabular-nums;">{hexv}</span></div></div>')
    palette = ('<div style="display: grid; grid-template-columns: repeat(8, minmax(0, 1fr)); gap: 12px;">'
               + sw_block("#eef1f6","--bg") + sw_block("#ffffff","--surface") + sw_block("#16202b","--ink") + sw_block("#5b6a7c","--muted")
               + sw_block("#ffd27a","--on-light") + sw_block("#9fe3d2","--on-air") + sw_block("#c9d7ff","--on-curtain") + sw_block("#ffc9b8","--on-power")
               + sw_block("#0f1318","--bg (dark)") + sw_block("#1a2029","--surface (dark)") + sw_block("#eef1f6","--ink (dark)") + sw_block("#8b97a7","--muted (dark)")
               + sw_block("#d9f5ea","--ok-bg") + sw_block("#1f7a55","--ok") + sw_block("#fde2e3","--danger-bg") + sw_block("#b3261e","--danger") + '</div>')
    typo = ('<div style="display: flex; flex-direction: column; gap: 8px;">'
            '<div class="display" style="font-size: 26px; font-weight: 600;">Unbounded 600 · 26 — заголовки комнат и экранов</div>'
            '<div class="display" style="font-size: 20px; font-weight: 600;">Unbounded 600 · 20 — шапка «Умный дом»</div>'
            '<div style="font-size: 34px; font-weight: 600; letter-spacing: -.02em;">Onest 600 · 34 — крупные показания 24,6</div>'
            '<div style="font-size: 15px; font-weight: 600;">Onest 600 · 15 — название устройства</div>'
            '<div style="font-size: 14px;">Onest 400 · 14 — основной текст, кнопки 500</div>'
            '<div style="font-size: 13px;" class="muted">Onest 400 · 13 — состояние устройства, подписи слайдеров</div>'
            '<div style="font-size: 12px; font-weight: 600;">Onest 600 · 12 — бейджи</div></div>')
    buttons = ('<div class="row" style="flex-wrap: wrap; gap: 12px;"><span class="btn primary">Сохранить</span><span class="btn">Выключить всё</span><span class="btn danger">Удалить</span>'
               f'<span class="btn primary disabled">Войти</span><span class="icon-btn">{svg("REFRESH",18)}</span><span class="icon-btn" style="color: #e5484d;">{svg("TRASH",16)}</span>'
               '<span class="tab active">Вкладка</span><span class="tab">Вкладка</span><span class="chip active">Спальня <span style="opacity: .6;">14</span></span><span class="chip">Кабинет <span style="opacity: .6;">9</span></span></div>')
    controls = ('<div style="display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 24px; align-items: start;">'
                '<div style="display: flex; flex-direction: column; gap: 14px;"><div class="row"><div class="sw on"></div><span class="muted">вкл</span><div class="sw"></div><span class="muted">выкл</span><div class="sw" style="opacity: .45;"></div><span class="muted">disabled</span></div>'
                + slider("Яркость",64,"64 %") + slider("Тепло",35,"3900 K") + '</div>'
                '<div style="display: flex; flex-direction: column; gap: 14px;"><div class="sel" style="background: #ffffff;"><span>Скорость</span><span class="row" style="gap: 4px;">турбо ' + CHEV() + '</span></div>' + swatches(2) +
                '<div class="row"><span class="badge ok">' + OK() + '</span><span class="badge busy">Запускаем…</span><span class="badge err">не отвечает</span></div></div>'
                '<div style="display: flex; flex-direction: column; gap: 14px;"><div class="input"><span class="ph">Код подтверждения</span></div><div class="input focus">1234567</div>'
                '<div class="banner">Яндекс ответил ошибкой: токен недействителен — войдите заново</div></div></div>')
    tiles = ('<div style="display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 14px;">'
             + tile("LAMP","Лампа выключена","выключена · 3400 K","",False)
             + tile("LAMP","Лампа включена","включена · 3400 K","on-light",True)
             + tile("PLUG","Розетка включена","включена · 0,25 Вт","on-power",True)
             + tile("AIR","Климат работает","работает · 51 %","on-air",True)
             + tile("CURTAIN","Шторы открыты","открыты на 20 %","on-curtain",True)
             + tile("DROP","Применяем",'<span class="badge busy">применяем…</span>',"busy",True)
             + tile("PLUG","Ошибка","Питание: устройство не отвечает","",False,"","err") + '</div>')
    def block(title, inner):
        return f'<section style="display: flex; flex-direction: column; gap: 12px;"><h3 class="display" style="margin: 0; font-size: 16px; font-weight: 600;">{title}</h3>{inner}</section>'
    return ('<div style="width: 1280px; height: 1240px; overflow: hidden; background: #eef1f6; display: flex; flex-direction: column; gap: 28px; padding: 26px 32px; box-sizing: border-box;">'
            '<div class="display" style="font-size: 20px; font-weight: 600;">Система «Плитки» — токены и компоненты</div>'
            + block("Палитра · светлая и тёмная", palette) + block("Типографика", typo) + block("Кнопки, вкладки, чипы", buttons)
            + block("Переключатели, слайдер, селект, цвета, бейджи, поля, баннер", controls) + block("Плитка устройства — состояния", tiles) + '</div>')

(D / "Main.dc.html").write_text(doc(devices_body(False)))
(D / "DevicesDark.dc.html").write_text(doc(devices_body(True), dark=True))
(D / "Login.dc.html").write_text(doc(login_body()))
(D / "Scenarios.dc.html").write_text(doc(scenarios_body()))
(D / "MacroEditor.dc.html").write_text(doc(editor_body()))
(D / "Mobile.dc.html").write_text(doc(mobile_body()))
(D / "Components.dc.html").write_text(doc(components_body()))
print("built 7 artboards")
