// 计划选项保留服务端提供的时长值和时区标识。
const intervalChoices = [
  { value: '1m', count: 1, unit: 'minutes' },
  { value: '5m', count: 5, unit: 'minutes' },
  { value: '15m', count: 15, unit: 'minutes' },
  { value: '30m', count: 30, unit: 'minutes' },
  { value: '1h', count: 1, unit: 'hours' },
  { value: '2h', count: 2, unit: 'hours' },
  { value: '6h', count: 6, unit: 'hours' },
  { value: '12h', count: 12, unit: 'hours' },
  { value: '24h', count: 1, unit: 'days' },
  { value: '48h', count: 2, unit: 'days' },
  { value: '168h', count: 7, unit: 'days' },
] as const

const durationUnits: Record<string, number> = {
  ns: 1e-9,
  us: 1e-6,
  µs: 1e-6,
  μs: 1e-6,
  ms: 1e-3,
  s: 1,
  m: 60,
  h: 3600,
}

// Go durations can contain multiple units, including a trailing zero component.
function durationSeconds(value: string): number | null {
  const duration = value.trim()
  const parts = [
    ...duration.matchAll(/(\d+(?:\.\d*)?|\.\d+)(ns|us|µs|μs|ms|s|m|h)/g),
  ]
  if (!parts.length || parts.map((part) => part[0]).join('') !== duration) {
    return null
  }
  return parts.reduce(
    (total, part) => total + Number(part[1]) * durationUnits[part[2]],
    0,
  )
}

export function getIntervalOptions(currentValue: string) {
  const currentSeconds = durationSeconds(currentValue)
  const matching = intervalChoices.find(
    (choice) => durationSeconds(choice.value) === currentSeconds,
  )
  const options: { value: string; count?: number; unit?: string }[] = [
    ...intervalChoices,
  ]
  if (currentValue && !matching) options.push({ value: currentValue })
  return { options, value: matching?.value ?? currentValue }
}

// 时区列表来自运行时数据，已保存的别名继续作为原值提供给选择控件。
export function getTimeZoneOptions(currentValue: string): string[] {
  const zones = Intl.supportedValuesOf?.('timeZone') ?? []
  return [
    ...new Set(
      [
        ...zones,
        'UTC',
        Intl.DateTimeFormat().resolvedOptions().timeZone,
        currentValue,
      ].filter(Boolean),
    ),
  ].sort((left, right) => left.localeCompare(right))
}
