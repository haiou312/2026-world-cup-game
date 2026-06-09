<script setup lang="ts">
import { computed } from 'vue'
import type { Fixture } from '@/types'

const props = defineProps<{ fixture: Fixture | null }>()

function side(which: 'home' | 'away') {
  const f = props.fixture
  const t = f ? f[which] : null
  const known = !!(t && t.team_id != null)
  return {
    name: known ? t!.name ?? 'TBD' : t?.seed ?? 'TBD',
    flag: known ? t!.flag_url : null,
    score: f ? (which === 'home' ? f.home_score : f.away_score) : null,
    win: !!(f && known && f.winner_team_id === t!.team_id),
    out: !!(known && t!.eliminated),
    players: known ? t!.players ?? [] : [],
    known,
  }
}

const home = computed(() => side('home'))
const away = computed(() => side('away'))
const empty = computed(() => !home.value.known && !away.value.known)
</script>

<template>
  <div class="cell" :class="{ empty }">
    <div class="trow" :class="{ win: home.win, out: home.out, seed: !home.known }">
      <img v-if="home.flag" :src="home.flag" class="f" />
      <span class="nm" :title="home.players.join(', ')">{{ home.name }}</span>
      <span v-if="home.score != null" class="sc">{{ home.score }}</span>
    </div>
    <div class="trow" :class="{ win: away.win, out: away.out, seed: !away.known }">
      <img v-if="away.flag" :src="away.flag" class="f" />
      <span class="nm" :title="away.players.join(', ')">{{ away.name }}</span>
      <span v-if="away.score != null" class="sc">{{ away.score }}</span>
    </div>
  </div>
</template>

<style scoped>
.cell {
  width: 104px;
  background: var(--panel-2);
  border: 1px solid var(--border);
  border-radius: 7px;
  overflow: hidden;
}
.cell.empty { background: #161620; border-style: dashed; }
.trow {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 3px 6px;
  font-size: 12px;
}
.trow + .trow { border-top: 1px solid var(--border); }
.trow .f { width: 16px; height: 11px; object-fit: cover; border-radius: 2px; }
.trow .nm { flex: 1; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.trow .sc { font-weight: 800; min-width: 12px; text-align: right; }
.trow.seed .nm { color: var(--muted); font-weight: 700; }
.trow.win .nm { color: var(--green); }
.trow.win { background: rgba(52, 199, 123, 0.12); }
.trow.out { opacity: 0.4; }
.trow.out .nm { text-decoration: line-through; }
.trow.out .f { filter: grayscale(1); }
</style>
