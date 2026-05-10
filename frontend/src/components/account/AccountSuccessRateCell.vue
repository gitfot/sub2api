<template>
  <div v-if="props.loading && !props.stats" class="space-y-1">
    <div class="h-3 w-14 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
    <div class="h-3 w-20 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
  </div>

  <div v-else-if="props.error && !props.stats" class="text-xs text-gray-400">--</div>

  <div v-else-if="props.stats" class="space-y-0.5 text-xs">
    <div class="font-medium text-gray-900 dark:text-white">
      {{ formatPercent(props.stats.success_rate) }}
    </div>
    <div class="text-gray-500 dark:text-gray-400">
      {{ formatCount(props.stats.success_count) }} / {{ formatCount(props.stats.failed_count) }}
    </div>
  </div>

  <div v-else class="text-xs text-gray-400">--</div>
</template>

<script setup lang="ts">
import type { SuccessRateSummary } from '@/types'

const props = withDefaults(
  defineProps<{
    stats?: SuccessRateSummary | null
    loading?: boolean
    error?: string | null
  }>(),
  {
    stats: null,
    loading: false,
    error: null
  }
)

const formatCount = (value: number): string => {
  return value.toLocaleString()
}

const formatPercent = (value: number | null | undefined): string => {
  if (value === null || value === undefined) return '--'
  return `${value.toFixed(2)}%`
}
</script>
