<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api/client'
import { useGate } from '@/stores/gate'
import type { ParticipantsResp, Participant, Assignment } from '@/types'
import SpinReel from '@/components/SpinReel.vue'

interface Team {
  id: number
  name: string
  code: string | null
  flag_url: string | null
  group_label: string | null
}

const gate = useGate()
const pw = ref('')
const gateErr = ref(false)
const unlocking = ref(false)

const participants = ref<Participant[]>([])
const teams = ref<Team[]>([])
const loadErr = ref('')

const personReel = ref<InstanceType<typeof SpinReel> | null>(null)
const countryReel = ref<InstanceType<typeof SpinReel> | null>(null)

const currentPerson = ref<Participant | null>(null)
const resultTeam = ref<Team | null>(null)
const step = ref<'idle' | 'person' | 'assigned'>('idle')
const spinning = ref(false)
const assignErr = ref('')

const unassigned = computed(() => participants.value.filter((p) => !p.assigned))
const assignedCount = computed(() => participants.value.filter((p) => p.assigned).length)
const currentRound = computed(() => Math.floor(assignedCount.value / 48))
const available = computed(() => {
  const used = new Set(
    participants.value.filter((p) => p.assigned && p.round === currentRound.value).map((p) => p.team_id),
  )
  return teams.value.filter((t) => !used.has(t.id))
})

const personItems = computed(() => unassigned.value.map((p) => ({ key: p.id, label: p.name })))
const countryItems = computed(() => available.value.map((t) => ({ key: t.id, label: t.name, flag: t.flag_url })))

async function unlock() {
  gateErr.value = false
  unlocking.value = true
  const ok = await gate.unlock(pw.value)
  unlocking.value = false
  if (ok) await load()
  else gateErr.value = true
}

async function load() {
  loadErr.value = ''
  try {
    const [p, t] = await Promise.all([
      api.get<ParticipantsResp>('/participants'),
      api.get<{ teams: Team[] }>('/teams'),
    ])
    participants.value = p.participants
    teams.value = t.teams
  } catch (e) {
    loadErr.value = (e as Error).message
  }
}

async function spinPerson() {
  if (spinning.value || !unassigned.value.length) return
  resultTeam.value = null
  assignErr.value = ''
  spinning.value = true
  const idx = Math.floor(Math.random() * unassigned.value.length)
  const person = unassigned.value[idx]
  await personReel.value?.spinTo(idx)
  currentPerson.value = person
  step.value = 'person'
  spinning.value = false
}

async function spinCountry() {
  if (spinning.value || step.value !== 'person' || !currentPerson.value) return
  spinning.value = true
  assignErr.value = ''
  try {
    const r = await api.post<{ assignment: Assignment }>('/assign', { participant_id: currentPerson.value.id })
    const teamId = r.assignment.team_id
    const idx = available.value.findIndex((t) => t.id === teamId)
    await countryReel.value?.spinTo(idx >= 0 ? idx : 0)
    resultTeam.value =
      available.value[idx] ?? {
        id: teamId,
        name: r.assignment.team_name,
        code: null,
        flag_url: r.assignment.flag_url,
        group_label: null,
      }
    step.value = 'assigned'
    await load() // refresh both pools
  } catch (e) {
    assignErr.value = (e as Error).message
  } finally {
    spinning.value = false
  }
}

onMounted(() => {
  if (gate.unlocked) load()
})
</script>

<template>
  <div class="container">
    <h2 class="title">{{ $t('wheel.title') }}</h2>

    <!-- password gate -->
    <div v-if="!gate.unlocked" class="card gate">
      <p class="muted">{{ $t('wheel.locked') }}</p>
      <div class="field">
        <label>{{ $t('settings.password') }}</label>
        <input v-model="pw" type="password" @keyup.enter="unlock" />
      </div>
      <button class="primary" :disabled="unlocking || !pw" @click="unlock">{{ $t('wheel.unlock') }}</button>
      <p v-if="gateErr" class="error">{{ $t('wheel.wrongPw') }}</p>
    </div>

    <template v-else>
      <p v-if="loadErr" class="error">{{ loadErr }}</p>

      <div class="wheels">
        <div class="wheel">
          <div class="wlabel">{{ $t('wheel.person') }}</div>
          <SpinReel ref="personReel" :items="personItems" />
          <button class="primary spin" :disabled="spinning || !unassigned.length" @click="spinPerson">
            {{ $t('wheel.spinPerson') }}
          </button>
        </div>
        <div class="wheel">
          <div class="wlabel">{{ $t('wheel.country') }}</div>
          <SpinReel ref="countryReel" :items="countryItems" />
          <button class="primary spin" :disabled="spinning || step !== 'person'" @click="spinCountry">
            {{ $t('wheel.spinCountry') }}
          </button>
        </div>
      </div>

      <div v-if="currentPerson" class="card result" :class="{ done: step === 'assigned' }">
        <span class="rp">{{ currentPerson.name }}</span>
        <span class="arrow">→</span>
        <template v-if="resultTeam">
          <img v-if="resultTeam.flag_url" :src="resultTeam.flag_url" class="rf" />
          <span class="rt">{{ resultTeam.name }}</span>
        </template>
        <span v-else class="muted">{{ $t('wheel.spinCountryHint') }}</span>
      </div>
      <p v-if="assignErr" class="error">{{ assignErr }}</p>

      <div class="counts muted">
        {{ $t('wheel.remainingPeople', { n: unassigned.length }) }} ·
        {{ $t('wheel.remainingCountries', { n: available.length }) }}
      </div>
    </template>
  </div>
</template>

<style scoped>
.gate { max-width: 360px; }
.wheels { display: grid; grid-template-columns: 1fr 1fr; gap: 18px; margin: 6px 0 18px; }
.wlabel { text-align: center; font-weight: 800; color: var(--gold); margin-bottom: 8px; letter-spacing: 0.5px; }
.spin { width: 100%; margin-top: 12px; }
.result {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  font-size: 20px;
  font-weight: 800;
}
.result.done { border: 2px solid var(--gold); box-shadow: 0 0 22px rgba(231, 184, 78, 0.22); }
.result .rp { color: var(--text); }
.result .arrow { color: var(--muted); }
.result .rt { color: var(--gold); }
.result .rf { width: 34px; height: 22px; object-fit: cover; border-radius: 3px; }
.counts { text-align: center; margin-top: 14px; font-size: 13px; }
@media (max-width: 640px) {
  .wheels { grid-template-columns: 1fr; }
}
</style>
