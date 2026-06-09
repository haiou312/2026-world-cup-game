<script setup lang="ts">
import type { Group, GroupTeam } from '@/types'
import { groupColors } from '@/lib/bracket2026'
import { computed } from 'vue'

const props = defineProps<{ group: Group }>()
const color = computed(() => groupColors[props.group.group] ?? '#888')

function statusClass(t: GroupTeam): string {
  if (t.adv_status === 'advancing') return 'adv'
  if (t.adv_status === 'third') {
    if (t.third_decided === false) return 'third-tbd'
    return t.third_qualifying ? 'third-in' : 'third-out'
  }
  if (t.adv_status === 'out') return 'out'
  return ''
}
function gd(t: GroupTeam): string {
  const v = t.goal_diff ?? 0
  return v > 0 ? `+${v}` : `${v}`
}
</script>

<template>
  <div class="gs" :style="{ '--g': color }">
    <div class="gs-head">GROUP {{ group.group }}</div>
    <table>
      <thead>
        <tr><th class="p">#</th><th class="t">Team</th><th>Pld</th><th>GD</th><th>Pts</th></tr>
      </thead>
      <tbody>
        <tr v-for="(t, i) in group.teams" :key="t.team_id" :class="statusClass(t)">
          <td class="p">{{ t.position || i + 1 }}</td>
          <td class="t">
            <img v-if="t.flag_url" :src="t.flag_url" class="flag" />
            <span class="nm">{{ t.name }}</span>
            <span v-if="t.adv_status === 'third'" class="badge3" :class="{ in: t.third_qualifying && t.third_decided }">
              {{ t.third_decided === false ? '3rd' : (t.third_qualifying ? 'in' : 'out') }}
            </span>
          </td>
          <td>{{ t.played ?? 0 }}</td>
          <td>{{ gd(t) }}</td>
          <td class="pts">{{ t.points ?? 0 }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.gs {
  background: linear-gradient(180deg, var(--panel), var(--bg-2));
  border: 1px solid var(--border);
  border-top: 3px solid var(--g);
  border-radius: 10px;
  overflow: hidden;
}
.gs-head { font-weight: 800; color: var(--g); padding: 8px 10px 4px; font-size: 13px; }
table { width: 100%; border-collapse: collapse; font-size: 12px; }
th, td { padding: 4px 6px; text-align: center; }
th { color: var(--muted); font-weight: 600; font-size: 11px; }
td:first-child, th:first-child { width: 18px; }
.t { text-align: left; }
td.t { display: flex; align-items: center; gap: 6px; }
.flag { width: 18px; height: 12px; object-fit: cover; border-radius: 2px; }
.nm { font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.pts { font-weight: 800; }
tbody tr { border-top: 1px solid var(--border); }
/* status: left accent + tint */
tbody tr.adv { box-shadow: inset 3px 0 0 var(--green); }
tbody tr.third-tbd { box-shadow: inset 3px 0 0 #f59e0b; }
tbody tr.third-in { box-shadow: inset 3px 0 0 #f59e0b; }
tbody tr.third-out { box-shadow: inset 3px 0 0 #f59e0b; opacity: 0.6; }
tbody tr.out { opacity: 0.45; }
.badge3 {
  font-size: 9px;
  padding: 0 5px;
  border-radius: 999px;
  border: 1px solid #f59e0b;
  color: #f59e0b;
}
.badge3.in { background: #f59e0b; color: #1a1205; }
</style>
