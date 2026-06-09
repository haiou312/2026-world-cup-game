<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api/client'
import { stageLabel } from '@/lib/stage'
import type { ParticipantsResp, Participant, Fixture } from '@/types'
import MatchRow from '@/components/MatchRow.vue'

const data = ref<ParticipantsResp | null>(null)
const error = ref('')
const busy = ref(false)
const search = ref('')
const selected = ref<Participant | null>(null)
const matches = ref<Fixture[]>([])
const detailBusy = ref(false)

const filtered = computed(() => {
  const list = data.value?.participants ?? []
  const q = search.value.trim().toLowerCase()
  return q ? list.filter((p) => p.name.toLowerCase().includes(q)) : list
})

async function load() {
  busy.value = true
  error.value = ''
  try {
    data.value = await api.get<ParticipantsResp>('/participants')
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}

async function openDetail(p: Participant) {
  selected.value = p
  matches.value = []
  if (!p.assigned || !p.team_id) return
  detailBusy.value = true
  try {
    const fx = await api.get<{ fixtures: Fixture[] }>(`/fixtures?team=${p.team_id}`)
    matches.value = fx.fixtures
  } catch {
    /* fixtures are best-effort */
  } finally {
    detailBusy.value = false
  }
}
function back() {
  selected.value = null
  matches.value = []
}

onMounted(load)
</script>

<template>
  <div class="container">
    <!-- DETAIL: one participant's "My Team" -->
    <template v-if="selected">
      <button class="back" @click="back">← {{ $t('home.back') }}</button>
      <div class="card team-card">
        <div class="row" style="gap: 14px">
          <img v-if="selected.flag_url" class="flag big" :src="selected.flag_url" />
          <div>
            <div class="pname">{{ selected.name }}</div>
            <div v-if="selected.assigned" style="font-size: 22px; font-weight: 800">{{ selected.team_name }}</div>
            <div class="muted" v-if="selected.assigned">
              {{ $t('home.groupRound', { group: selected.group_label || '?', round: (selected.round ?? 0) + 1 }) }}
            </div>
            <div class="muted" v-else>{{ $t('home.notAssigned') }}</div>
          </div>
          <span style="flex: 1" />
          <span v-if="selected.assigned && selected.champion" class="tag champ">{{ $t('home.champion') }}</span>
          <span v-else-if="selected.assigned && selected.eliminated" class="tag out">{{ $t('home.out', { stage: stageLabel(selected.furthest_stage || 'GROUP') }) }}</span>
          <span v-else-if="selected.assigned" class="tag alive">{{ $t('home.alive', { stage: stageLabel(selected.furthest_stage || 'GROUP') }) }}</span>
        </div>
      </div>

      <template v-if="selected.assigned">
        <h3 class="muted sec">{{ $t('home.fixtures') }}</h3>
        <div class="card matches">
          <MatchRow v-for="f in matches" :key="f.id" :fixture="f" />
          <p v-if="detailBusy" class="muted empty">{{ $t('common.loading') }}</p>
          <p v-else-if="!matches.length" class="muted empty">{{ $t('home.noFixtures') }}</p>
        </div>
      </template>
    </template>

    <!-- LIST + search -->
    <template v-else>
      <div class="row" style="margin-bottom: 14px">
        <h2 class="title" style="margin: 0">{{ $t('home.title') }}</h2>
        <span style="flex: 1" />
        <button @click="load" :disabled="busy">{{ busy ? $t('common.refreshing') : '↻ ' + $t('common.refresh') }}</button>
      </div>

      <p v-if="busy && !data" class="muted state">{{ $t('common.loading') }}</p>

      <div v-else-if="error && !data" class="card state">
        <p class="error" style="margin: 0 0 10px">{{ $t('home.loadError') }}</p>
        <p class="muted" style="margin: 0 0 12px; font-size: 13px">{{ error }}</p>
        <button @click="load">{{ $t('common.retry') }}</button>
      </div>

      <template v-else-if="data">
        <div class="card stat" style="margin-bottom: 14px">
          <span class="seg alive">{{ $t('home.statAlive', { n: data.remaining }) }}</span>
          <span class="dot">·</span>
          <span class="seg">{{ $t('home.statDrawn', { n: data.assigned }) }}</span>
          <template v-if="data.unassigned">
            <span class="dot">·</span>
            <span class="seg dim">{{ $t('home.statNotDrawn', { n: data.unassigned }) }}</span>
          </template>
        </div>

        <input class="search" v-model="search" :placeholder="$t('home.searchPlaceholder')" />

        <p v-if="!data.total" class="muted state">{{ $t('home.empty') }}</p>
        <p v-else-if="!filtered.length" class="muted state">{{ $t('home.noMatch') }}</p>
        <div v-else class="list">
          <button
            v-for="p in filtered"
            :key="p.id"
            class="p-row card"
            :class="{ out: p.eliminated, champ: p.champion, off: !p.assigned }"
            @click="openDetail(p)"
          >
            <img v-if="p.flag_url" class="flag" :src="p.flag_url" />
            <span v-else class="flag ph" />
            <div class="who">
              <div class="u">{{ p.name }}</div>
              <div class="muted t">{{ p.assigned ? p.team_name : $t('home.notAssigned') }}</div>
            </div>
            <span style="flex: 1" />
            <span v-if="p.champion" class="tag champ">{{ $t('home.champion') }}</span>
            <span v-else-if="p.assigned && p.eliminated" class="tag out">{{ $t('home.out', { stage: stageLabel(p.furthest_stage || 'GROUP') }) }}</span>
            <span v-else-if="p.assigned" class="tag alive">{{ stageLabel(p.furthest_stage || 'GROUP') }}</span>
          </button>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.state { padding: 30px 14px; text-align: center; }
.stat { display: flex; align-items: baseline; gap: 8px; flex-wrap: wrap; font-size: 15px; }
.seg.alive { color: var(--green); font-weight: 800; }
.seg.dim { color: var(--muted); }
.dot { color: var(--muted); }
.back { margin-bottom: 12px; }
.pname { font-size: 13px; color: var(--muted); font-weight: 700; }
.flag.big { width: 46px; height: 30px; object-fit: cover; border-radius: 4px; }
.sec { margin: 24px 0 10px; }
.matches { padding: 0; overflow: hidden; }
.matches .empty { padding: 14px; margin: 0; }
.search {
  width: 100%;
  margin-bottom: 14px;
  padding: 10px 12px;
  font-size: 14px;
  box-sizing: border-box;
}
.list { display: flex; flex-direction: column; gap: 8px; }
.p-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  width: 100%;
  text-align: left;
  cursor: pointer;
}
.p-row:hover { border-color: var(--gold); }
.p-row.out { opacity: 0.55; }
.p-row.off { opacity: 0.7; }
.p-row.champ { border-color: var(--accent); }
.flag { width: 26px; height: 17px; object-fit: cover; border-radius: 2px; }
.flag.ph { background: var(--panel-2); }
.who .u { font-weight: 700; }
.who .t { font-size: 12px; }
</style>
