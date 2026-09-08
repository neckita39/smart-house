import { useEffect, useRef, useState } from 'react'
import { eventLabel, label, modeLabel, unit } from './labels'
import { hexToHsv, hexToRgbInt, hsvToHex, rgbIntToHex } from './color'

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
      return <Switch text="Питание" checked={!!state.value} disabled={busy} onChange={(v) => onChange(cap.type, 'on', v)} />
    case 'toggle':
      return <Switch text={label(p.instance)} checked={!!state.value} disabled={busy} onChange={(v) => onChange(cap.type, p.instance, v)} />
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

function Switch({ text, checked, disabled, onChange }) {
  const [local, setLocal] = useSynced(checked)
  return (
    <div className="control">
      <label>{text}</label>
      <input
        type="checkbox"
        checked={local}
        disabled={disabled}
        onChange={(e) => {
          const prev = local
          setLocal(e.target.checked)
          onChange(e.target.checked).then((ok) => {
            if (!ok) setLocal(prev)
          })
        }}
      />
      <span className="value">{local ? 'вкл' : 'выкл'}</span>
    </div>
  )
}

function Slider({ text, min, max, step, value, suffix, disabled, onCommit }) {
  const [local, setLocal] = useSynced(value)
  const commit = () => {
    if (local === value) return
    const prev = value
    onCommit(Number(local)).then((ok) => {
      if (!ok) setLocal(prev)
    })
  }
  return (
    <div className="control">
      <label>{text}</label>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={local}
        disabled={disabled}
        onChange={(e) => setLocal(Number(e.target.value))}
        onPointerUp={commit}
        onKeyUp={commit}
      />
      <span className="value">
        {local}
        {suffix}
      </span>
    </div>
  )
}

function Range({ cap, busy, onChange }) {
  const p = cap.parameters || {}
  const r = p.range || { min: 0, max: 100, precision: 1 }
  return (
    <Slider
      text={label(p.instance)}
      min={r.min}
      max={r.max}
      step={r.precision || 1}
      value={cap.state?.value ?? r.min}
      suffix={unit(p.unit)}
      disabled={busy}
      onCommit={(v) => onChange(cap.type, p.instance, v)}
    />
  )
}

function Mode({ cap, busy, onChange }) {
  const p = cap.parameters || {}
  return (
    <div className="control">
      <label>{label(p.instance)}</label>
      <select value={cap.state?.value ?? ''} disabled={busy} onChange={(e) => onChange(cap.type, p.instance, e.target.value)}>
        <option value="" disabled>
          —
        </option>
        {(p.modes || []).map((m) => (
          <option key={m.value} value={m.value}>
            {modeLabel(m.value)}
          </option>
        ))}
      </select>
    </div>
  )
}

function ColorSetting({ cap, busy, onChange }) {
  const p = cap.parameters || {}
  const state = cap.state || {}
  const model = p.color_model // 'hsv' | 'rgb' | undefined
  const current = model && state.instance === model ? state.value : null
  const hex = current == null ? '#ffffff' : model === 'rgb' ? rgbIntToHex(current) : hsvToHex(current)
  const [localHex, setLocalHex] = useSynced(hex)
  const commitColor = useDebounced((h) => {
    const prev = hex
    onChange(cap.type, model, model === 'rgb' ? hexToRgbInt(h) : hexToHsv(h)).then((ok) => {
      if (!ok) setLocalHex(prev)
    })
  })

  return (
    <>
      {p.temperature_k && (
        <Slider
          text="Цвет. температура"
          min={p.temperature_k.min}
          max={p.temperature_k.max}
          step={100}
          value={state.instance === 'temperature_k' ? state.value : p.temperature_k.min}
          suffix=" K"
          disabled={busy}
          onCommit={(v) => onChange(cap.type, 'temperature_k', v)}
        />
      )}
      {model && (
        <div className="control">
          <label>Цвет</label>
          <input
            type="color"
            value={localHex}
            disabled={busy}
            onChange={(e) => {
              setLocalHex(e.target.value)
              commitColor(e.target.value)
            }}
          />
        </div>
      )}
      {p.color_scene?.scenes?.length > 0 && (
        <div className="control">
          <label>Сцена</label>
          <select
            value={state.instance === 'scene' ? state.value : ''}
            disabled={busy}
            onChange={(e) => onChange(cap.type, 'scene', e.target.value)}
          >
            <option value="">—</option>
            {p.color_scene.scenes.map((s) => (
              <option key={s.id} value={s.id}>
                {s.id}
              </option>
            ))}
          </select>
        </div>
      )}
    </>
  )
}

// PropertyReadout — показание датчика: число с единицей или событие.
export function PropertyReadout({ prop }) {
  const p = prop.parameters || {}
  const v = prop.state?.value
  const kind = prop.type.split('.').pop() // 'float' | 'event'
  let text
  if (v == null) text = '—'
  else if (kind === 'event') text = eventLabel(v)
  else text = `${typeof v === 'number' ? Math.round(v * 10) / 10 : v} ${unit(p.unit)}`.trim()
  return (
    <span title={p.instance}>
      <span className="muted">{label(p.instance)}: </span>
      {text}
    </span>
  )
}
