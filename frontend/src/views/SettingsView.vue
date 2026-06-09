<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '@/api/client'
import { useGate } from '@/stores/gate'
import type { ParticipantsResp, Participant, SyncStatus } from '@/types'

const { t } = useI18n()
const gate = useGate()
const pw = ref('')
const gateErr = ref(false)
const unlocking = ref(false)

const participants = ref<Participant[]>([])
const newName = ref('')
const status = ref<SyncStatus | null>(null)
const newKey = ref('')
const msg = ref('')
const error = ref('')
const busy = ref(false)

async function unlock() {
  gateErr.value = false
  unlocking.value = true
  const ok = await gate.unlock(pw.value)
  unlocking.value = false
  if (ok) await loadAll()
  else gateErr.value = true
}

async function loadAll() {
  error.value = ''
  try {
    const [p, s] = await Promise.all([
      api.get<ParticipantsResp>('/participants'),
      api.get<SyncStatus>('/settings/sync-status'),
    ])
    participants.value = p.participants
    status.value = s
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function addParticipant() {
  const name = newName.value.trim()
  if (!name) return
  error.value = ''
  msg.value = ''
  try {
    await api.post('/participants', { name })
    newName.value = ''
    await loadAll()
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function removeParticipant(p: Participant) {
  error.value = ''
  try {
    await api.del(`/participants/${p.id}`)
    await loadAll()
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function resetAll() {
  if (!confirm(t('settings.resetConfirm'))) return
  error.value = ''
  try {
    await api.post('/reset')
    await loadAll()
    msg.value = t('settings.resetDone')
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function saveKey() {
  msg.value = ''
  error.value = ''
  busy.value = true
  try {
    await api.put('/settings/api-key', { key: newKey.value.trim() })
    newKey.value = ''
    msg.value = t('settings.tokenSaved')
    await loadAll()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}

async function manualSync() {
  msg.value = ''
  error.value = ''
  busy.value = true
  try {
    status.value = await api.post<SyncStatus>('/settings/sync')
    msg.value = t('settings.syncComplete')
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}

onMounted(() => {
  if (gate.unlocked) loadAll()
})
</script>

<template>
  <div class="container">
    <h2 class="title">{{ $t('nav.settings') }}</h2>

    <!-- password gate -->
    <div v-if="!gate.unlocked" class="card gate">
      <p class="muted">{{ $t('settings.locked') }}</p>
      <div class="field">
        <label>{{ $t('settings.password') }}</label>
        <input v-model="pw" type="password" @keyup.enter="unlock" />
      </div>
      <button class="primary" :disabled="unlocking || !pw" @click="unlock">{{ $t('settings.unlock') }}</button>
      <p v-if="gateErr" class="error">{{ $t('settings.wrongPw') }}</p>
    </div>

    <template v-else>
      <!-- participants -->
      <div class="card" style="margin-bottom: 16px">
        <h3 style="margin-top: 0">{{ $t('settings.participants') }} ({{ participants.length }})</h3>
        <div class="row" style="gap: 8px; margin-bottom: 12px">
          <input v-model="newName" :placeholder="$t('settings.newName')" @keyup.enter="addParticipant" style="flex: 1" />
          <button class="primary" :disabled="!newName.trim()" @click="addParticipant">{{ $t('settings.add') }}</button>
        </div>
        <p v-if="!participants.length" class="muted">{{ $t('settings.noParticipants') }}</p>
        <table v-else class="ptable">
          <thead>
            <tr>
              <th class="idx">#</th>
              <th>{{ $t('settings.colName') }}</th>
              <th>{{ $t('settings.colTeam') }}</th>
              <th class="act"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(p, i) in participants" :key="p.id">
              <td class="idx">{{ i + 1 }}</td>
              <td class="nm">{{ p.name }}</td>
              <td>
                <div v-if="p.assigned" class="teamcell">
                  <img v-if="p.flag_url" :src="p.flag_url" class="tflag" />
                  <span>{{ p.team_name }}</span>
                </div>
                <span v-else class="muted">{{ $t('settings.unassigned') }}</span>
              </td>
              <td class="act">
                <button class="del" @click="removeParticipant(p)" :title="$t('settings.remove')">✕</button>
              </td>
            </tr>
          </tbody>
        </table>
        <button class="reset" @click="resetAll" style="margin-top: 12px">{{ $t('settings.resetDraws') }}</button>
      </div>

      <!-- token -->
      <div class="card" style="margin-bottom: 16px">
        <h3 style="margin-top: 0">{{ $t('admin.tokenTitle') }}</h3>
        <p class="muted">
          {{ $t('admin.status') }}
          <span v-if="status?.api_key_configured" class="tag alive">{{ $t('admin.configured', { masked: status.api_key_masked }) }}</span>
          <span v-else class="tag out">{{ $t('admin.notConfigured') }}</span>
        </p>
        <div class="field">
          <label>{{ $t('admin.enterToken') }}</label>
          <input v-model="newKey" placeholder="X-Auth-Token" />
        </div>
        <button class="primary" :disabled="busy || !newKey.trim()" @click="saveKey">{{ $t('admin.saveToken') }}</button>
      </div>

      <!-- sync -->
      <div class="card">
        <div class="row">
          <h3 style="margin: 0">{{ $t('admin.syncStatus') }}</h3>
          <span v-if="status" class="tag" :class="status.last_error ? 'out' : 'alive'">
            {{ status.last_error ? $t('admin.failed') : $t('admin.healthy') }}
          </span>
          <span style="flex: 1" />
          <button :disabled="busy" @click="manualSync">{{ $t('admin.syncNow') }}</button>
        </div>
        <table v-if="status" class="kv">
          <tbody>
            <tr><td>{{ $t('admin.lastSynced') }}</td><td>{{ status.last_synced_at || '—' }}</td></tr>
            <tr><td>{{ $t('admin.lastSuccess') }}</td><td>{{ status.last_success_at || '—' }}</td></tr>
            <tr v-if="status.last_error">
              <td>{{ $t('admin.lastError') }}</td>
              <td class="bad">{{ status.last_error }}</td>
            </tr>
            <tr v-if="status.last_warnings"><td>{{ $t('admin.warnings') }}</td><td class="warn">{{ status.last_warnings }}</td></tr>
            <tr><td>{{ $t('admin.todayComplete') }}</td><td>{{ status.today_done ? $t('admin.yes') : $t('admin.no') }}</td></tr>
            <tr><td>{{ $t('admin.apiCalls') }}</td><td>{{ status.api_calls_today }}</td></tr>
          </tbody>
        </table>
        <p v-if="msg" class="muted" style="margin-top: 10px">{{ msg }}</p>
        <p v-if="error" class="error">{{ error }}</p>
      </div>
    </template>
  </div>
</template>

<style scoped>
.gate { max-width: 360px; }
.ptable { width: 100%; border-collapse: collapse; font-size: 14px; }
.ptable th {
  text-align: left;
  color: var(--muted);
  font-weight: 600;
  font-size: 12px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--border);
}
.ptable td { padding: 8px; border-bottom: 1px solid var(--border); vertical-align: middle; }
.ptable .idx { width: 28px; text-align: right; color: var(--muted); }
.ptable .nm { font-weight: 700; }
.ptable .teamcell { display: flex; align-items: center; gap: 7px; }
.ptable .tflag { width: 22px; height: 14px; object-fit: cover; border-radius: 2px; }
.ptable .act { width: 40px; text-align: right; }
.del { padding: 2px 9px; font-size: 12px; color: #f87171; }
.reset { color: #f87171; border-color: #f87171; }
.kv { width: 100%; border-collapse: collapse; margin-top: 10px; }
.kv td { padding: 7px 4px; border-top: 1px solid var(--border); font-size: 14px; }
.kv td:first-child { color: var(--muted); width: 40%; }
.kv td.bad { color: #f87171; font-weight: 600; }
.kv td.warn { color: #f59e0b; }
</style>
