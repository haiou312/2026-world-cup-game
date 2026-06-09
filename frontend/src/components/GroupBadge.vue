<script setup lang="ts">
import { computed } from 'vue'
import type { Group } from '@/types'
import { groupColors } from '@/lib/bracket2026'

const props = defineProps<{ group: Group }>()
const color = computed(() => groupColors[props.group.group] ?? '#888')
</script>

<template>
  <div class="badge" :style="{ '--g': color }">
    <div class="flags">
      <div
        v-for="t in group.teams"
        :key="t.team_id"
        class="cell"
        :class="[t.adv_status, { dim: t.eliminated || t.adv_status === 'out', champ: t.champion }]"
        :title="`${t.name}${t.players.length ? ' — ' + t.players.join(', ') : ''}`"
      >
        <img v-if="t.flag_url" :src="t.flag_url" :alt="t.name" />
        <span v-else class="code">{{ t.code || t.name.slice(0, 3) }}</span>
      </div>
    </div>
    <div class="label">GROUP {{ group.group }}</div>
  </div>
</template>

<style scoped>
.badge {
  background: #0f0f12;
  border: 2px solid var(--g);
  border-radius: 14px;
  padding: 8px 8px 6px;
  box-shadow: 0 0 14px color-mix(in srgb, var(--g) 35%, transparent);
}
.flags {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 5px;
}
.cell {
  aspect-ratio: 4 / 3;
  border-radius: 5px;
  overflow: hidden;
  background: #222;
  display: flex;
  align-items: center;
  justify-content: center;
}
.cell img { width: 100%; height: 100%; object-fit: cover; }
.cell .code { font-size: 11px; font-weight: 700; color: var(--muted); }
.cell.advancing { outline: 2px solid var(--green); outline-offset: -2px; }
.cell.third { outline: 2px solid #f59e0b; outline-offset: -2px; }
.cell.dim { opacity: 0.35; filter: grayscale(1); }
.cell.champ { outline: 2px solid var(--gold); outline-offset: -2px; }
.label {
  margin-top: 6px;
  text-align: center;
  font-weight: 800;
  font-size: 13px;
  letter-spacing: 0.5px;
  color: var(--g);
}
</style>
