<script setup lang="ts">
import { computed } from 'vue'
import type { Fixture } from '@/types'
import { stageLabel } from '@/lib/stage'

const props = defineProps<{ fixture: Fixture; compactDate?: boolean }>()

const finished = computed(() => ['FT', 'AET', 'PEN'].includes(props.fixture.status))
const live = computed(() => ['LIVE', 'HT', '1H', '2H', 'ET', 'P'].includes(props.fixture.status))

const label = computed(() => {
  const f = props.fixture
  if (f.stage === 'GROUP') return `Group ${f.group_label || '?'}`
  return stageLabel(f.stage)
})

function fmtWhen(d: string | null): string {
  if (!d) return 'TBD'
  const dt = new Date(d)
  return dt.toLocaleString(
    'en-GB',
    props.compactDate
      ? { hour: '2-digit', minute: '2-digit', hour12: false }
      : { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false },
  )
}
function won(teamId: number | null): boolean {
  return teamId != null && props.fixture.winner_team_id === teamId
}
function players(side: { players?: string[] }): string {
  return side.players?.length ? side.players.join(', ') : ''
}
</script>

<template>
  <div class="mrow">
    <div class="when">
      <span class="t">{{ fmtWhen(fixture.kickoff_at) }}</span>
      <span class="stg">{{ label }}</span>
    </div>

    <div class="side home" :class="{ win: won(fixture.home.team_id) }">
      <span class="nm" :title="players(fixture.home)">{{ fixture.home.name || 'TBD' }}</span>
      <img v-if="fixture.home.flag_url" :src="fixture.home.flag_url" class="flag" />
    </div>

    <div class="score" :class="{ live }">
      <template v-if="fixture.home_score != null">{{ fixture.home_score }} : {{ fixture.away_score }}</template>
      <template v-else>vs</template>
    </div>

    <div class="side away" :class="{ win: won(fixture.away.team_id) }">
      <img v-if="fixture.away.flag_url" :src="fixture.away.flag_url" class="flag" />
      <span class="nm" :title="players(fixture.away)">{{ fixture.away.name || 'TBD' }}</span>
    </div>

    <div class="st" :class="{ done: finished, live }">
      {{ finished ? 'FT' : live ? 'LIVE' : 'Upcoming' }}
    </div>
  </div>
</template>

<style scoped>
.mrow {
  display: grid;
  grid-template-columns: 104px 1fr 64px 1fr 52px;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-top: 1px solid var(--border);
}
.when { display: flex; flex-direction: column; white-space: nowrap; }
.when .t { font-size: 13px; font-weight: 600; }
.when .stg { font-size: 11px; color: var(--gold); }
.side { display: flex; align-items: center; gap: 7px; min-width: 0; }
.side.home { justify-content: flex-end; }
.side.away { justify-content: flex-start; }
.side .nm { font-weight: 600; font-size: 14px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.side .flag { width: 24px; height: 16px; object-fit: cover; border-radius: 3px; }
.side.win .nm { color: var(--green); }
.score { text-align: center; font-weight: 800; font-size: 15px; }
.score.live { color: var(--green); }
.st { text-align: right; font-size: 12px; color: var(--muted); }
.st.done { color: var(--gold); }
.st.live { color: var(--green); }
</style>
