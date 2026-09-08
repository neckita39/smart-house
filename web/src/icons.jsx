// Иконки дизайна «Плитки» — контурные, 24×24, цвет наследуется через currentColor.
// Пути перенесены из design/canvas/_icons.txt.
import { shortType } from './labels'

const PATHS = {
  LAMP: (
    <>
      <path d="M9 18h6" />
      <path d="M10 21h4" />
      <path d="M12 3a6 6 0 0 0-4 10.5c.6.6 1 1.4 1 2.5h6c0-1.1.4-1.9 1-2.5A6 6 0 0 0 12 3z" />
    </>
  ),
  AIR: (
    <>
      <path d="M4 8h10a3 3 0 1 0-3-3" />
      <path d="M4 12h14a3 3 0 1 1-3 3" />
      <path d="M4 16h8" />
    </>
  ),
  CURTAIN: (
    <>
      <path d="M4 4h16" />
      <path d="M6 4v16" />
      <path d="M18 4v16" />
      <path d="M6 12c3 0 4 2 6 2s3-2 6-2" />
    </>
  ),
  DROP: <path d="M12 3s6 6.5 6 11a6 6 0 0 1-12 0c0-4.5 6-11 6-11z" />,
  FAN: (
    <>
      <circle cx="12" cy="12" r="2" />
      <path d="M12 10V4a4 4 0 0 1 4 4" />
      <path d="M14 12h6a4 4 0 0 1-4 4" />
      <path d="M12 14v6a4 4 0 0 1-4-4" />
      <path d="M10 12H4a4 4 0 0 1 4-4" />
    </>
  ),
  THERMO: <path d="M14 14.8V5a2 2 0 1 0-4 0v9.8a4 4 0 1 0 4 0z" />,
  TV: (
    <>
      <rect x="3" y="5" width="18" height="12" rx="2" />
      <path d="M8 21h8" />
    </>
  ),
  CAM: (
    <>
      <path d="M4 8h11l3 3h2v7H4z" />
      <circle cx="10" cy="14" r="2.5" />
    </>
  ),
  PLUG: (
    <>
      <rect x="4" y="4" width="16" height="16" rx="4" />
      <circle cx="9.5" cy="12" r="1" />
      <circle cx="14.5" cy="12" r="1" />
    </>
  ),
  REFRESH: (
    <>
      <path d="M20 12a8 8 0 1 1-2.3-5.7" />
      <path d="M20 4v5h-5" />
    </>
  ),
  PLAY: <path d="M7 5v14l11-7z" />,
  PLUS: (
    <>
      <path d="M12 5v14" />
      <path d="M5 12h14" />
    </>
  ),
  X: (
    <>
      <path d="M6 6l12 12" />
      <path d="M18 6L6 18" />
    </>
  ),
  PEN: <path d="M4 20l4-1 11-11-3-3L5 16z" />,
  TRASH: (
    <>
      <path d="M4 7h16" />
      <path d="M9 7V4h6v3" />
      <path d="M6 7l1 13h10l1-13" />
    </>
  ),
  CHECK: <path d="M5 12l5 5L20 7" />,
  CHEV: <path d="M6 9l6 6 6-6" />,
}

// Icon рисует иконку по имени. stroke — толщина линии (в макетах 1.8, у мелких — 2 и 2.4).
export function Icon({ name, size = 20, stroke = 1.8 }) {
  const paths = PATHS[name]
  if (!paths) return null
  return (
    <svg
      viewBox="0 0 24 24"
      width={size}
      height={size}
      fill="none"
      stroke="currentColor"
      strokeWidth={stroke}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      {paths}
    </svg>
  )
}

// Соответствие «тип устройства → иконка и тинт включённого состояния».
// Порядок важен: сначала более частные типы. tint = null — плитка не тонируется.
const KINDS = [
  [/^light/, 'LAMP', 'on-light'],
  [/^sensor\.climate/, 'AIR', null],
  [/^sensor/, 'PLUG', null],
  [/^thermostat/, 'THERMO', 'on-air'],
  [/^humidifier/, 'DROP', 'on-air'],
  [/^ventilation/, 'FAN', 'on-air'],
  [/^purifier/, 'AIR', 'on-air'],
  [/^openable/, 'CURTAIN', 'on-curtain'],
  [/^media_device/, 'TV', 'on-power'],
  [/^camera/, 'CAM', 'on-power'],
]

function kind(type) {
  const short = shortType(type)
  return KINDS.find(([re]) => re.test(short)) || [null, 'PLUG', 'on-power']
}

export const deviceIcon = (type) => kind(type)[1]
export const deviceTint = (type) => kind(type)[2]
