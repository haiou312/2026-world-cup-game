import { describe, it, expect } from 'vitest'
import { stageLabel } from '@/lib/stage'
import { setLocale } from '@/i18n'

describe('stageLabel', () => {
  it('maps stage codes to English labels', () => {
    setLocale('en')
    expect(stageLabel('GROUP')).toBe('Group Stage')
    expect(stageLabel('R32')).toBe('Round of 32')
    expect(stageLabel('FINAL')).toBe('Final')
  })

  it('switches with the locale', () => {
    setLocale('zh')
    expect(stageLabel('R32')).toBe('32 强')
    expect(stageLabel('FINAL')).toBe('决赛')
    setLocale('en')
  })

  it('passes unknown codes through unchanged', () => {
    expect(stageLabel('WEIRD')).toBe('WEIRD')
  })
})
