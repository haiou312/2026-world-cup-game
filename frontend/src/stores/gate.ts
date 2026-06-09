import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, setSettingsPw } from '@/api/client'

// Gate guards the host-only Settings + Wheel sections behind the shared
// password. Kept in memory only, so it must be re-entered each session.
export const useGate = defineStore('gate', () => {
  const unlocked = ref(false)

  async function unlock(pw: string): Promise<boolean> {
    setSettingsPw(pw)
    try {
      await api.post('/settings/verify')
      unlocked.value = true
      return true
    } catch {
      setSettingsPw('')
      unlocked.value = false
      return false
    }
  }

  function lock() {
    unlocked.value = false
    setSettingsPw('')
  }

  return { unlocked, unlock, lock }
})
