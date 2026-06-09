import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SpinReel from '@/components/SpinReel.vue'

const items3 = [
  { key: 1, label: 'A' },
  { key: 2, label: 'B' },
  { key: 3, label: 'C' },
]

describe('SpinReel', () => {
  it('renders every item into the spin strip', () => {
    const w = mount(SpinReel, { props: { items: items3 } })
    const cells = w.findAll('.cell')
    expect(cells.length).toBeGreaterThan(items3.length)
    expect(cells.length % items3.length).toBe(0)
    expect(w.text()).toContain('A')
    expect(w.text()).toContain('C')
  })

  it('snaps back to the top when items change (the blank-wheel fix)', async () => {
    const w = mount(SpinReel, { props: { items: items3 } })

    // start a spin and let the two animation frames fire so it scrolls down
    ;(w.vm as unknown as { spinTo: (i: number) => Promise<void> }).spinTo(2)
    await new Promise((r) => setTimeout(r, 90))
    await w.vm.$nextTick()
    const scrolled = w.find('.strip').attributes('style') || ''
    expect(scrolled).toMatch(/translateY\(-\d/) // scrolled down (non-zero)

    // remove an item → the watcher resets the reel to the top, keeping the
    // remaining items visible instead of scrolled off-screen
    await w.setProps({ items: [{ key: 1, label: 'A' }] })
    const reset = w.find('.strip').attributes('style') || ''
    expect(reset).toContain('translateY(0px)')
  })
})
