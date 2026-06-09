import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import en from '@/i18n/en'

vi.mock('@/api/client', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), del: vi.fn() },
  setSettingsPw: vi.fn(),
}))
import { api } from '@/api/client'
import HomeView from '@/views/HomeView.vue'

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en } })

function mountHome() {
  return mount(HomeView, { global: { plugins: [i18n], stubs: { MatchRow: true } } })
}

describe('HomeView search', () => {
  beforeEach(() => {
    ;(api.get as ReturnType<typeof vi.fn>).mockReset()
  })

  it('filters the participant list by name (case-insensitive)', async () => {
    ;(api.get as ReturnType<typeof vi.fn>).mockResolvedValue({
      total: 3,
      assigned: 0,
      unassigned: 3,
      remaining: 3,
      participants: [
        { id: 1, name: 'Alice', assigned: false },
        { id: 2, name: 'Bob', assigned: false },
        { id: 3, name: 'Alicia', assigned: false },
      ],
    })
    const w = mountHome()
    await flushPromises()

    expect(w.findAll('.p-row').length).toBe(3)

    await w.find('.search').setValue('ali')
    const names = w.findAll('.p-row .u').map((n) => n.text())
    expect(names).toEqual(['Alice', 'Alicia'])

    await w.find('.search').setValue('zzz')
    expect(w.findAll('.p-row').length).toBe(0)
  })
})
