<script setup lang="ts">
import { computed } from 'vue'
import type { Round, Fixture, GroupTeam } from '@/types'
import KnockoutCell from './KnockoutCell.vue'

const props = defineProps<{ rounds: Round[]; champion: GroupTeam | null }>()

interface Cell {
  fixture: Fixture | null
}

function fxOf(stage: string): Fixture[] {
  return props.rounds.find((r) => r.stage === stage)?.fixtures ?? []
}
function halves(stage: string): [Fixture[], Fixture[]] {
  const fx = fxOf(stage)
  const mid = Math.ceil(fx.length / 2)
  return [fx.slice(0, mid), fx.slice(mid)]
}
function pad(fx: Fixture[], count: number): Cell[] {
  const cells: Cell[] = fx.slice(0, count).map((f) => ({ fixture: f }))
  while (cells.length < count) cells.push({ fixture: null })
  return cells
}

const left = computed(() => ({
  r32: pad(halves('R32')[0], 8),
  r16: pad(halves('R16')[0], 4),
  qf: pad(halves('QF')[0], 2),
  sf: pad(halves('SF')[0], 1),
}))
const right = computed(() => ({
  r32: pad(halves('R32')[1], 8),
  r16: pad(halves('R16')[1], 4),
  qf: pad(halves('QF')[1], 2),
  sf: pad(halves('SF')[1], 1),
}))
const finalCell = computed<Cell>(() => ({ fixture: fxOf('FINAL')[0] ?? null }))
</script>

<template>
  <div class="kb">
    <!-- LEFT half: R32 → R16 → QF → SF, converging right -->
    <div class="side left">
      <div class="round">
        <div class="cells">
          <div v-for="(c, i) in left.r32" :key="i" class="slot"><KnockoutCell v-bind="c" /></div>
        </div>
      </div>
      <div class="round">
        <div class="cells">
          <div v-for="(c, i) in left.r16" :key="i" class="slot"><KnockoutCell v-bind="c" /></div>
        </div>
      </div>
      <div class="round">
        <div class="cells">
          <div v-for="(c, i) in left.qf" :key="i" class="slot"><KnockoutCell v-bind="c" /></div>
        </div>
      </div>
      <div class="round last">
        <div class="cells">
          <div v-for="(c, i) in left.sf" :key="i" class="slot"><KnockoutCell v-bind="c" /></div>
        </div>
      </div>
    </div>

    <!-- CENTER: final + champion -->
    <div class="center">
      <div class="ctitle">{{ $t('bracket.worldChampions') }}</div>
      <div class="champ" :class="{ won: champion }">
        <template v-if="champion">
          <img v-if="champion.flag_url" :src="champion.flag_url" class="cflag" />
          <div class="cname">{{ champion.name }}</div>
          <div class="cplayers">{{ champion.players.join(', ') }}</div>
        </template>
        <img v-else src="/trophy.png" alt="Trophy" class="trophy-img" />
      </div>
      <div class="final-slot">
        <KnockoutCell v-bind="finalCell" />
      </div>
      <div class="cfoot">FIFA WORLD CUP 2026</div>
    </div>

    <!-- RIGHT half: SF → QF → R16 → R32, mirrored -->
    <div class="side right">
      <div class="round last">
        <div class="cells">
          <div v-for="(c, i) in right.sf" :key="i" class="slot"><KnockoutCell v-bind="c" /></div>
        </div>
      </div>
      <div class="round">
        <div class="cells">
          <div v-for="(c, i) in right.qf" :key="i" class="slot"><KnockoutCell v-bind="c" /></div>
        </div>
      </div>
      <div class="round">
        <div class="cells">
          <div v-for="(c, i) in right.r16" :key="i" class="slot"><KnockoutCell v-bind="c" /></div>
        </div>
      </div>
      <div class="round">
        <div class="cells">
          <div v-for="(c, i) in right.r32" :key="i" class="slot"><KnockoutCell v-bind="c" /></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.kb {
  --ln: rgba(244, 240, 230, 0.32);
  --gap: 16px;
  display: flex;
  align-items: stretch;
  justify-content: safe center;
  overflow-x: auto;
  padding: 8px 0 18px;
}
.side { display: flex; align-items: stretch; }
.round { display: flex; flex-direction: column; }
.rlabel {
  text-align: center;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.4px;
  color: var(--gold);
  height: 22px;
}
.cells { flex: 1; display: flex; flex-direction: column; }
.slot {
  flex: 1;
  min-height: 60px;
  display: flex;
  align-items: center;
  position: relative;
}

/* ===== LEFT half connectors (point right toward center) ===== */
.side.left .round { margin-right: var(--gap); }
.side.left .round:not(.last) .slot::after {
  content: '';
  position: absolute;
  left: 100%;
  width: calc(var(--gap) / 2);
  border-right: 2px solid var(--ln);
}
.side.left .round:not(.last) .slot:nth-child(odd)::after {
  top: 50%;
  height: 50%;
  border-top: 2px solid var(--ln);
}
.side.left .round:not(.last) .slot:nth-child(even)::after {
  bottom: 50%;
  height: 50%;
  border-bottom: 2px solid var(--ln);
}
.side.left .round:not(:first-child) .slot::before {
  content: '';
  position: absolute;
  right: 100%;
  width: calc(var(--gap) / 2);
  top: 50%;
  height: 2px;
  background: var(--ln);
}
.side.left .round.last .slot::after {
  content: '';
  position: absolute;
  left: 100%;
  width: var(--gap);
  top: 50%;
  height: 2px;
  background: var(--ln);
}

/* ===== RIGHT half connectors (mirrored, point left) ===== */
.side.right .round { margin-left: var(--gap); }
.side.right .round:not(.last) .slot::after {
  content: '';
  position: absolute;
  right: 100%;
  width: calc(var(--gap) / 2);
  border-left: 2px solid var(--ln);
}
.side.right .round:not(.last) .slot:nth-child(odd)::after {
  top: 50%;
  height: 50%;
  border-top: 2px solid var(--ln);
}
.side.right .round:not(.last) .slot:nth-child(even)::after {
  bottom: 50%;
  height: 50%;
  border-bottom: 2px solid var(--ln);
}
.side.right .round:not(:last-child) .slot::before {
  content: '';
  position: absolute;
  left: 100%;
  width: calc(var(--gap) / 2);
  top: 50%;
  height: 2px;
  background: var(--ln);
}
.side.right .round.last .slot::after {
  content: '';
  position: absolute;
  right: 100%;
  width: var(--gap);
  top: 50%;
  height: 2px;
  background: var(--ln);
}

/* ===== center ===== */
.center {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 0 14px;
  min-width: 150px;
}
.ctitle { font-weight: 900; letter-spacing: 1px; font-size: 16px; text-align: center; margin-bottom: 10px; }
.champ {
  width: 140px;
  height: 64px;
  border-radius: 12px;
  background: #161620;
  border: 1px dashed var(--border);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
}
.champ.won {
  border: 2px solid var(--gold);
  background: linear-gradient(180deg, rgba(231, 184, 78, 0.18), rgba(231, 184, 78, 0.04));
  box-shadow: 0 0 26px rgba(231, 184, 78, 0.25);
}
.champ .trophy-img { height: 46px; width: auto; filter: drop-shadow(0 4px 10px rgba(231, 184, 78, 0.45)); }
.cflag { width: 32px; height: 21px; object-fit: cover; border-radius: 3px; }
.cname { font-weight: 800; color: var(--gold); }
.cplayers { font-size: 11px; color: var(--muted); }
.final-slot { margin-top: 16px; display: flex; flex-direction: column; align-items: center; gap: 4px; }
.center-label { color: var(--gold); }
.cfoot { margin-top: 14px; font-size: 11px; color: var(--muted); letter-spacing: 1px; }
</style>
