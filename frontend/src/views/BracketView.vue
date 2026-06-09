<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { Bracket, Group, Fixture } from '@/types'
import { topGroups, bottomGroups } from '@/lib/bracket2026'
import { stageLabel } from '@/lib/stage'
import GroupBadge from '@/components/GroupBadge.vue'
import GroupStandings from '@/components/GroupStandings.vue'
import KnockoutBracket from '@/components/KnockoutBracket.vue'
import MatchRow from '@/components/MatchRow.vue'

const STAGES = ['GROUP', 'R32', 'R16', 'QF', 'SF', 'THIRD', 'FINAL']

const data = ref<Bracket | null>(null)
const allFixtures = ref<Fixture[]>([])
const selectedStage = ref<'ALL' | string>('ALL')
const error = ref('')
const busy = ref(false)

const byGroup = computed<Record<string, Group>>(() => {
  const m: Record<string, Group> = {}
  for (const g of data.value?.groups ?? []) m[g.group] = g
  return m
})
const topGroupObjs = computed(() => topGroups.map((g) => byGroup.value[g]).filter(Boolean) as Group[])
const bottomGroupObjs = computed(() => bottomGroups.map((g) => byGroup.value[g]).filter(Boolean) as Group[])

const presentStages = computed(() => {
  const set = new Set(allFixtures.value.map((f) => f.stage))
  return STAGES.filter((s) => set.has(s))
})

// Fixtures (filtered by stage chip) grouped by local calendar date, in
// chronological order. Knockout dates naturally follow group dates, so the
// schedule "advances" into the knockout rounds on its own as the days pass.
const dateGroups = computed(() => {
  const filtered =
    selectedStage.value === 'ALL'
      ? allFixtures.value
      : allFixtures.value.filter((f) => f.stage === selectedStage.value)

  const todayKey = new Date().toLocaleDateString('en-GB')
  const order: string[] = []
  const map = new Map<string, Fixture[]>()
  for (const f of filtered) {
    const key = f.kickoff_at ? new Date(f.kickoff_at).toLocaleDateString('en-GB') : 'TBD'
    if (!map.has(key)) {
      map.set(key, [])
      order.push(key)
    }
    map.get(key)!.push(f)
  }

  return order.map((key) => {
    const matches = map.get(key)!
    const first = matches[0].kickoff_at
    const label = first
      ? new Date(first).toLocaleDateString('en-GB', { weekday: 'short', month: 'short', day: 'numeric' })
      : 'TBD'
    return { key, label, isToday: key === todayKey, matches }
  })
})

async function load() {
  busy.value = true
  error.value = ''
  try {
    const [b, fx] = await Promise.all([
      api.get<Bracket>('/bracket'),
      api.get<{ fixtures: Fixture[] }>('/fixtures'),
    ])
    data.value = b
    allFixtures.value = fx.fixtures
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="container wide">
    <div class="row" style="margin-bottom: 16px">
      <h2 class="title" style="margin: 0">{{ $t('bracket.title') }}</h2>
      <span style="flex: 1" />
      <button @click="load" :disabled="busy">{{ busy ? $t('common.refreshing') : '↻ ' + $t('common.refresh') }}</button>
    </div>

    <p v-if="busy && !data" class="muted bstate">{{ $t('common.loading') }}</p>

    <div v-else-if="error && !data" class="card bstate">
      <p class="error" style="margin: 0 0 10px">{{ $t('bracket.loadError') }}</p>
      <p class="muted" style="margin: 0 0 12px; font-size: 13px">{{ error }}</p>
      <button @click="load">{{ $t('common.retry') }}</button>
    </div>

    <template v-else-if="data">
      <p v-if="error" class="error">{{ error }}</p>
      <p v-if="!data.groups.length && !allFixtures.length" class="muted bstate">{{ $t('bracket.empty') }}</p>

      <template v-else>
        <div class="badge-row">
          <GroupBadge v-for="g in topGroupObjs" :key="g.group" :group="g" />
        </div>

        <KnockoutBracket :rounds="data.rounds" :champion="data.champion" />

        <div class="badge-row">
          <GroupBadge v-for="g in bottomGroupObjs" :key="g.group" :group="g" />
        </div>

        <!-- group standings: live tables with advancement status -->
        <div class="row section">
          <h3 class="muted" style="margin: 0">{{ $t('bracket.groupStandings') }}</h3>
          <span class="legend">
            <i class="dot adv" /> {{ $t('bracket.advancing') }}
            <i class="dot third" /> {{ $t('bracket.thirdRace') }}
            <i class="dot out" /> {{ $t('bracket.out') }}
          </span>
        </div>
        <div class="standings-grid">
          <GroupStandings v-for="g in [...topGroupObjs, ...bottomGroupObjs]" :key="g.group" :group="g" />
        </div>

        <!-- schedule & scores: one date-ordered timeline -->
        <h3 class="muted section">{{ $t('bracket.fixturesScores') }}</h3>
        <div class="tabs">
          <button :class="{ active: selectedStage === 'ALL' }" @click="selectedStage = 'ALL'">{{ $t('bracket.all') }}</button>
          <button
            v-for="s in presentStages"
            :key="s"
            :class="{ active: selectedStage === s }"
            @click="selectedStage = s"
          >
            {{ stageLabel(s) }}
          </button>
        </div>

        <div class="card schedule">
          <div v-for="d in dateGroups" :key="d.key" class="day">
            <div class="day-head" :class="{ today: d.isToday }">
              {{ d.label }}<span v-if="d.isToday" class="today-badge">{{ $t('common.today') }}</span>
            </div>
            <MatchRow v-for="f in d.matches" :key="f.id" :fixture="f" compact-date />
          </div>
          <p v-if="!dateGroups.length" class="muted empty">{{ $t('bracket.noFixtures') }}</p>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.wide { max-width: 1240px; }
.bstate { padding: 30px 14px; text-align: center; }
.badge-row {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 12px;
  margin: 12px 0;
}
.section { margin: 30px 0 10px; align-items: baseline; gap: 14px; }
.legend { font-size: 12px; color: var(--muted); display: flex; align-items: center; gap: 6px; }
.legend .dot { width: 9px; height: 9px; border-radius: 50%; display: inline-block; margin-left: 8px; }
.legend .dot.adv { background: var(--green); }
.legend .dot.third { background: #f59e0b; }
.legend .dot.out { background: var(--muted); }
.standings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.tabs { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 14px; }
.tabs button { padding: 6px 12px; font-size: 13px; }
.tabs button.active { background: var(--gold); color: #1a1205; border-color: var(--gold); font-weight: 700; }
.schedule { padding: 0; overflow: hidden; }
.day-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: var(--panel-2);
  font-weight: 700;
  font-size: 13px;
  color: var(--text);
  border-top: 1px solid var(--border);
}
.day-head.today { color: var(--gold); }
.today-badge {
  font-size: 11px;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--gold);
  color: #1a1205;
  font-weight: 700;
}
.schedule .empty { padding: 14px; margin: 0; }
@media (max-width: 900px) {
  .badge-row { grid-template-columns: repeat(3, 1fr); }
}
</style>
