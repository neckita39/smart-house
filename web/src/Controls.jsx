import { useEffect, useRef, useState } from 'react'
import { eventLabel, formatNumber, label, modeLabel, rangeLabel, sceneLabel, unit } from './labels'
import { hexToHsv, hexToRgbInt, hsvToHex, rgbIntToHex } from './color'
import { Icon } from './icons'

// Пресеты цвета из макета: клик отправляет цвет сразу, свой цвет — нативной пипеткой.
const PRESETS = [
  ['#ffd27a', 'тёплый'],
  ['#ff9a6c', 'оранжевый'],
  ['#8fb7e8', 'голубой'],
  ['#9fe3d2', 'мятный'],
  ['#e9b7ff', 'сиреневый'],
  ['#ffffff', 'белый'],
]

// useSynced — локальное значение, которое подхватывает новое значение с сервера
// (оптимистичное обновление: показываем сразу, сервер подтвердит при следующем опросе).
function useSynced(value) {
  const [local, setLocal] = useState(value)
  useEffect(() => setLocal(value), [value])
  return [local, setLocal]
}

// useDebounced вызывает fn через delay мс после последнего вызова (для color picker,
// который сыплет события при каждом движении курсора).
function useDebounced(fn, delay = 400) {
  const timer = useRef(null)
  useEffect(() => () => clearTimeout(timer.current), [])
  return (...args) => {
    clearTimeout(timer.current)
    timer.current = setTimeout(() => fn(...args), delay)
  }
}

// Capability выбирает контрол по типу умения. onChange(type, instance, value).
export function Capability({ cap, busy, onChange }) {
  const kind = cap.type.split('.').pop()
  const p = cap.parameters || {}
  const state = cap.state || {}
  switch (kind) {
    case 'on_off':
      return <SwitchRow text="Питание" checked={!!state.value} disabled={busy} onChange={(v) => onChange(cap.type, 'on', v)} />
    case 'toggle':
      return <SwitchRow text={label(p.instance)} checked={!!state.value} disabled={busy} onChange={(v) => onChange(cap.type, p.instance, v)} />
    case 'range':
      return <Range cap={cap} busy={busy} onChange={onChange} />
    case 'mode':
      return <Mode cap={cap} busy={busy} onChange={onChange} />
    case 'color_setting':
      return <ColorSetting cap={cap} busy={busy} onChange={onChange} />
    default:
      return null
  }
}

// Switch — тумблер .sw из макета: настоящий чекбокс поверх нарисованной пилюли.
export function Switch({ text, checked, disabled, onChange, small }) {
  const [local, setLocal] = useSynced(checked)
  return (
    <span className={`sw${local ? ' on' : ''}${disabled ? ' disabled' : ''}${small ? ' small' : ''}`}>
      <input
        type="checkbox"
        checked={local}
        disabled={disabled}
        aria-label={text}
        onChange={(e) => {
          const prev = local
          setLocal(e.target.checked)
          onChange(e.target.checked).then((ok) => {
            if (!ok) setLocal(prev)
          })
        }}
      />
    </span>
  )
}

// SwitchRow — строка «подпись + маленький тумблер» для toggle-умений.
function SwitchRow({ text, checked, disabled, onChange }) {
  return (
    <div className="row between">
      <span className="state">{text}</span>
      <Switch text={text} checked={checked} disabled={disabled} onChange={onChange} small />
    </div>
  )
}

function Slider({ text, min, max, step, value, unitKey, disabled, onCommit }) {
  const [local, setLocal] = useSynced(value)
  const commit = () => {
    if (local === value) return
    const prev = value
    onCommit(Number(local)).then((ok) => {
      if (!ok) setLocal(prev)
    })
  }
  const pct = max > min ? Math.round(((Number(local) - min) / (max - min)) * 1000) / 10 : 0
  const u = unit(unitKey)
  return (
    <label className="slider">
      {text && <span className="lbl">{text}</span>}
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={local}
        disabled={disabled}
        aria-label={text}
        style={{ '--pct': `${pct}%` }}
        onChange={(e) => setLocal(Number(e.target.value))}
        onPointerUp={commit}
        onKeyUp={commit}
      />
      <span className="val">{`${formatNumber(Number(local), unitKey)}${u ? ' ' + u : ''}`}</span>
    </label>
  )
}

function Range({ cap, busy, onChange }) {
  const p = cap.parameters || {}
  const r = p.range || { min: 0, max: 100, precision: 1 }
  return (
    <Slider
      text={rangeLabel(p.instance)}
      min={r.min}
      max={r.max}
      step={r.precision || 1}
      value={cap.state?.value ?? r.min}
      unitKey={p.unit}
      disabled={busy}
      onCommit={(v) => onChange(cap.type, p.instance, v)}
    />
  )
}

// Select — нативный select внутри пилюли .sel с иконкой CHEV.
function Select({ text, value, disabled, onChange, children }) {
  return (
    <label className="sel">
      <span className="lbl">{text}</span>
      <span className="pick">
        <select value={value} disabled={disabled} aria-label={text} onChange={onChange}>
          {children}
        </select>
        <Icon name="CHEV" size={14} stroke={2} />
      </span>
    </label>
  )
}

function Mode({ cap, busy, onChange }) {
  const p = cap.parameters || {}
  const value = cap.state?.value ?? ''
  return (
    <Select
      text={label(p.instance)}
      value={value}
      disabled={busy}
      onChange={(e) => onChange(cap.type, p.instance, e.target.value)}
    >
      {!value && (
        <option value="" disabled>
          —
        </option>
      )}
      {(p.modes || []).map((m) => (
        <option key={m.value} value={m.value}>
          {modeLabel(m.value)}
        </option>
      ))}
    </Select>
  )
}

function ColorSetting({ cap, busy, onChange }) {
  const p = cap.parameters || {}
  const state = cap.state || {}
  const model = p.color_model // 'hsv' | 'rgb' | undefined
  const current = model && state.instance === model ? state.value : null
  const hex = current == null ? '#ffffff' : model === 'rgb' ? rgbIntToHex(current) : hsvToHex(current)
  const [localHex, setLocalHex] = useSynced(hex)
  // Свотч отмечен выбранным только когда устройство и правда светит цветом:
  // при температуре света или сцене выделять нечего.
  const [colorMode, setColorMode] = useSynced(current != null)
  const push = (h) => {
    const prevHex = hex
    const prevMode = current != null
    setColorMode(true)
    onChange(cap.type, model, model === 'rgb' ? hexToRgbInt(h) : hexToHsv(h)).then((ok) => {
      if (!ok) {
        setLocalHex(prevHex)
        setColorMode(prevMode)
      }
    })
  }
  const pushDebounced = useDebounced(push)
  const pick = (h) => {
    setLocalHex(h)
    push(h)
  }

  return (
    <>
      {p.temperature_k && (
        <Slider
          text="Тепло света"
          min={p.temperature_k.min}
          max={p.temperature_k.max}
          step={100}
          value={state.instance === 'temperature_k' ? state.value : p.temperature_k.min}
          unitKey="unit.temperature.kelvin"
          disabled={busy}
          onCommit={(v) => onChange(cap.type, 'temperature_k', v)}
        />
      )}
      {model && (
        <div className="swatches">
          {PRESETS.map(([c, name]) => (
            <button
              key={c}
              type="button"
              className={`swatch${colorMode && localHex.toLowerCase() === c ? ' sel' : ''}`}
              style={{ background: c }}
              disabled={busy}
              title={`Цвет: ${name}`}
              aria-label={`Цвет: ${name}`}
              onClick={() => pick(c)}
            />
          ))}
          <input
            type="color"
            className="swatch"
            value={localHex}
            disabled={busy}
            title="Свой цвет"
            aria-label="Свой цвет"
            onChange={(e) => {
              setLocalHex(e.target.value)
              pushDebounced(e.target.value)
            }}
          />
        </div>
      )}
      {p.color_scene?.scenes?.length > 0 && (
        <Select
          text="Сцена"
          value={state.instance === 'scene' ? state.value : ''}
          disabled={busy}
          onChange={(e) => onChange(cap.type, 'scene', e.target.value)}
        >
          <option value="">—</option>
          {p.color_scene.scenes.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name || sceneLabel(s.id)}
            </option>
          ))}
        </Select>
      )}
    </>
  )
}

// PropertyReadout — крупное показание датчика: число .big и единица .unit.
export function PropertyReadout({ prop }) {
  const p = prop.parameters || {}
  const v = prop.state?.value
  const text = v == null ? '—' : typeof v === 'number' ? formatNumber(v, p.unit) : eventLabel(v)
  return (
    <div className="readout" title={label(p.instance)}>
      <span className="big">{text}</span>
      <span className="unit">{unit(p.unit)}</span>
    </div>
  )
}
