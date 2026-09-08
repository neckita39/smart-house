export const hexToRgbInt = (hex) => parseInt(hex.slice(1), 16)

export const rgbIntToHex = (n) => '#' + (Number(n) & 0xffffff).toString(16).padStart(6, '0')

export function hexToHsv(hex) {
  const n = hexToRgbInt(hex)
  const r = ((n >> 16) & 255) / 255
  const g = ((n >> 8) & 255) / 255
  const b = (n & 255) / 255
  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  const d = max - min
  let h = 0
  if (d) {
    if (max === r) h = ((g - b) / d) % 6
    else if (max === g) h = (b - r) / d + 2
    else h = (r - g) / d + 4
    h = Math.round(h * 60)
    if (h < 0) h += 360
  }
  return { h, s: Math.round(max ? (d / max) * 100 : 0), v: Math.round(max * 100) }
}

export function hsvToHex({ h = 0, s = 0, v = 0 } = {}) {
  const S = s / 100
  const V = v / 100
  const c = V * S
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = V - c
  let rgb
  if (h < 60) rgb = [c, x, 0]
  else if (h < 120) rgb = [x, c, 0]
  else if (h < 180) rgb = [0, c, x]
  else if (h < 240) rgb = [0, x, c]
  else if (h < 300) rgb = [x, 0, c]
  else rgb = [c, 0, x]
  const [r, g, b] = rgb.map((q) => Math.round((q + m) * 255))
  return rgbIntToHex((r << 16) | (g << 8) | b)
}
