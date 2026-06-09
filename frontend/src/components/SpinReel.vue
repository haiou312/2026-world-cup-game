<script setup lang="ts">
import { computed, ref, watch } from 'vue'

interface Item {
  key: string | number
  label: string
  flag?: string | null
}
const props = defineProps<{ items: Item[] }>()

const ITEM_H = 46
const COPIES = 12
const offset = ref(0)
const dur = ref(0)

const strip = computed(() => {
  const out: Item[] = []
  for (let c = 0; c < COPIES; c++) out.push(...props.items)
  return out
})

// When the item list changes (e.g. a drawn person/country is removed), snap the
// reel back to the top so the remaining items stay visible instead of being
// scrolled past by a stale spin offset.
watch(
  () => props.items.length,
  () => {
    dur.value = 0
    offset.value = 0
  },
)

// spinTo animates the reel so items[index] lands in the centre slot.
function spinTo(index: number): Promise<void> {
  return new Promise((resolve) => {
    const n = props.items.length
    if (!n) {
      resolve()
      return
    }
    dur.value = 0
    offset.value = 0
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        const landCopy = COPIES - 2
        const target = (landCopy * n + index) * ITEM_H
        dur.value = 2.8
        offset.value = target - ITEM_H // centre row (of 3 visible)
        setTimeout(resolve, 2850)
      })
    })
  })
}
defineExpose({ spinTo })
</script>

<template>
  <div class="window">
    <div
      class="strip"
      :style="{
        transform: `translateY(${-offset}px)`,
        transition: dur ? `transform ${dur}s cubic-bezier(0.11,0.85,0.16,1)` : 'none',
      }"
    >
      <div v-for="(it, i) in strip" :key="i" class="cell">
        <img v-if="it.flag" :src="it.flag" class="rflag" />
        <span class="rlabel">{{ it.label }}</span>
      </div>
    </div>
    <div class="band" />
  </div>
</template>

<style scoped>
.window {
  position: relative;
  height: 138px; /* 3 rows */
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: linear-gradient(180deg, var(--panel), var(--bg-2));
}
.strip { will-change: transform; }
.cell {
  height: 46px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-weight: 700;
  padding: 0 10px;
  opacity: 0.5;
}
.rflag { width: 26px; height: 17px; object-fit: cover; border-radius: 2px; }
.rlabel { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.band {
  position: absolute;
  top: 46px;
  left: 0;
  right: 0;
  height: 46px;
  border-top: 2px solid var(--gold);
  border-bottom: 2px solid var(--gold);
  background: rgba(231, 184, 78, 0.12);
  pointer-events: none;
}
</style>
