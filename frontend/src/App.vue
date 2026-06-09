<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { setLocale, type Locale } from '@/i18n'

const { locale } = useI18n()

function toggleLang() {
  setLocale((locale.value === 'en' ? 'zh' : 'en') as Locale)
}
</script>

<template>
  <nav class="nav">
    <RouterLink to="/" class="brand">
      <img src="/wc-logo.png" alt="FIFA World Cup 26" class="logo" />
      <span>{{ $t('nav.brand') }}</span>
    </RouterLink>
    <RouterLink to="/">{{ $t('nav.home') }}</RouterLink>
    <RouterLink to="/bracket">{{ $t('nav.bracket') }}</RouterLink>
    <RouterLink to="/wheel">{{ $t('nav.wheel') }}</RouterLink>
    <RouterLink to="/settings">{{ $t('nav.settings') }}</RouterLink>
    <span class="spacer" />
    <button class="lang" @click="toggleLang">{{ locale === 'en' ? '中文' : 'EN' }}</button>
  </nav>
  <RouterView />
</template>

<style scoped>
.lang { font-size: 12px; padding: 4px 10px; }
</style>
