import { describe, expect, it, vi } from 'vitest'
import { getIntervalOptions, getTimeZoneOptions } from './scheduleOptions'

// Saved Go values must survive the transition from free text to selected choices.
describe('schedule options', () => {
  it.each([
    ['24h0m0s', '24h'],
    ['3600s', '1h'],
    ['0.5h', '30m'],
    ['60s', '1m'],
  ])('matches equivalent duration %s to %s', (saved, expected) => {
    const choices = getIntervalOptions(saved)
    expect(choices.value).toBe(expected)
    expect(choices.options).toHaveLength(11)
  })

  it('keeps nonpreset durations selectable without replacing the saved value', () => {
    const choices = getIntervalOptions('90m0s')
    expect(choices.value).toBe('90m0s')
    expect(choices.options.at(-1)).toEqual({ value: '90m0s' })
  })

  it('does not interpret partial or empty values as preset durations', () => {
    expect(getIntervalOptions('1h-invalid').value).toBe('1h-invalid')
    expect(getIntervalOptions('').options).toHaveLength(11)
  })

  it('retains saved aliases and UTC alongside runtime zones without duplicates', () => {
    const zones = getTimeZoneOptions('Asia/Calcutta')
    expect(zones).toContain('Asia/Calcutta')
    expect(zones).toContain('UTC')
    expect(zones).toContain(Intl.DateTimeFormat().resolvedOptions().timeZone)
    expect(new Set(zones).size).toBe(zones.length)
    expect(zones).toContain('Asia/Shanghai')
  })

  it('uses saved and local zones when runtime enumeration is unavailable', () => {
    const spy = vi.spyOn(Intl, 'supportedValuesOf').mockReturnValue([])
    try {
      expect(getTimeZoneOptions('Asia/Shanghai')).toContain('Asia/Shanghai')
      expect(
        getTimeZoneOptions('UTC').filter((zone) => zone === 'UTC'),
      ).toHaveLength(1)
    } finally {
      spy.mockRestore()
    }
  })
})
