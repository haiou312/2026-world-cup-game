import { i18n } from '@/i18n'

const known = ['GROUP', 'R32', 'R16', 'QF', 'SF', 'THIRD', 'FINAL']

// stageLabel resolves a stage code to its localized label. Reading the global
// i18n locale here keeps it reactive: templates re-render on language switch.
export function stageLabel(s: string): string {
  return known.includes(s) ? i18n.global.t(`stage.${s}`) : s
}
