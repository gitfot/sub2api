<template>
  <div class="card overflow-hidden border border-emerald-200/70 bg-gradient-to-br from-emerald-50 via-white to-cyan-50 p-4 shadow-sm dark:border-emerald-900/40 dark:from-emerald-950/30 dark:via-dark-900 dark:to-slate-950">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <div class="space-y-1">
        <div class="flex items-center gap-2">
          <div class="rounded-xl bg-emerald-500/10 p-2 text-emerald-700 ring-1 ring-inset ring-emerald-500/20 dark:text-emerald-300">
            <Icon name="badge" size="md" class="text-emerald-600 dark:text-emerald-300" :stroke-width="2" />
          </div>
          <div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.dashboard.successRateTrend') }}
            </h3>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.dashboard.requestSuccessRate') }}
              <span v-if="trendData?.computed_at" class="ml-2">
                · {{ trendData.computed_at }}
              </span>
            </p>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2 text-[11px] font-medium">
          <span
            v-if="trendData?.stale"
            class="rounded-full bg-amber-100 px-2 py-0.5 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
          >
            Stale
          </span>
          <span
            v-if="trendData?.partial"
            class="rounded-full bg-sky-100 px-2 py-0.5 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300"
          >
            Partial
          </span>
          <span
            v-if="latestPoint"
            class="rounded-full bg-emerald-100 px-2 py-0.5 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
          >
            {{ formatPercent(latestPoint.success_rate) }}
          </span>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <div class="flex items-center gap-2">
          <span class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('admin.dashboard.accountFilter') }}:
          </span>
          <div class="w-48">
            <Select
              :model-value="selectedAccountId"
              :options="accountOptions"
              @update:model-value="emit('update:selected-account-id', $event)"
            />
          </div>
        </div>
        <div class="inline-flex rounded-xl border border-emerald-200 bg-white/80 p-1 shadow-sm backdrop-blur dark:border-emerald-900/40 dark:bg-dark-900/70">
          <button
            v-for="option in granularityOptions"
            :key="option.value"
            type="button"
            class="rounded-lg px-3 py-1.5 text-xs font-medium transition-all"
            :class="granularity === option.value
              ? 'bg-emerald-600 text-white shadow-sm'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="emit('update:granularity', option.value)"
          >
            {{ option.label }}
          </button>
        </div>
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="loading"
          @click="emit('refresh')"
        >
          {{ t('common.refresh') }}
        </button>
      </div>
    </div>

    <div class="mb-4 grid gap-3 sm:grid-cols-3">
      <div class="rounded-2xl bg-white/80 p-3 shadow-sm ring-1 ring-inset ring-emerald-200/70 dark:bg-dark-900/60 dark:ring-emerald-900/30">
        <p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
          {{ t('admin.dashboard.todaySuccessRate') }}
        </p>
        <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">
          {{ formatPercent(latestPoint?.success_rate) }}
        </p>
      </div>
      <div class="rounded-2xl bg-white/80 p-3 shadow-sm ring-1 ring-inset ring-emerald-200/70 dark:bg-dark-900/60 dark:ring-emerald-900/30">
        <p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
          {{ t('admin.dashboard.successRateRequests') }}
        </p>
        <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">
          {{ formatNumber(totalRequests) }}
        </p>
      </div>
      <div class="rounded-2xl bg-white/80 p-3 shadow-sm ring-1 ring-inset ring-emerald-200/70 dark:bg-dark-900/60 dark:ring-emerald-900/30">
        <p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
          {{ t('admin.dashboard.historySuccessRate') }}
        </p>
        <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">
          {{ formatPercent(historySuccessRate) }}
        </p>
      </div>
    </div>

    <div v-if="loading" class="flex h-72 items-center justify-center">
      <LoadingSpinner />
    </div>
    <div v-else-if="chartData" class="h-72">
      <Bar :data="chartData" :options="chartOptions" />
    </div>
    <div
      v-else
      class="flex h-72 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
    >
      {{ t('admin.dashboard.noDataAvailable') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  PointElement,
  LineElement,
  Tooltip,
  Legend
} from 'chart.js'
import { Bar } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AccountSuccessRateTrendResponse } from '@/types'

ChartJS.register(CategoryScale, LinearScale, BarElement, PointElement, LineElement, Tooltip, Legend)

const { t } = useI18n()

type Granularity = '10m' | '1h' | '1d'

const props = withDefaults(defineProps<{
  trendData: AccountSuccessRateTrendResponse | null
  loading?: boolean
  granularity: Granularity
  accountOptions: Array<{ value: number; label: string }>
  selectedAccountId: number
}>(), {
  loading: false,
  accountOptions: () => [],
  selectedAccountId: 0
})

const emit = defineEmits<{
  refresh: []
  'update:granularity': [value: Granularity]
  'update:selected-account-id': [value: number | string | boolean | null]
}>()

const granularityOptions: Array<{ value: Granularity; label: string }> = [
  { value: '10m', label: t('admin.dashboard.successRateGranularity10m') },
  { value: '1h', label: t('admin.dashboard.successRateGranularity1h') },
  { value: '1d', label: t('admin.dashboard.successRateGranularity1d') }
]

const latestPoint = computed(() => {
  const points = props.trendData?.points || []
  return points.length > 0 ? points[points.length - 1] : null
})

const historySuccessRate = computed(() => {
  const points = props.trendData?.points || []
  if (!points.length) return null
  const totalSuccess = points.reduce((sum, point) => sum + point.success_count, 0)
  const totalRequests = points.reduce((sum, point) => sum + point.request_count, 0)
  return totalRequests > 0 ? (totalSuccess / totalRequests) * 100 : null
})

const totalRequests = computed(() => {
  return (props.trendData?.points || []).reduce((sum, point) => sum + point.request_count, 0)
})

const chartData = computed<any>(() => {
  const points = props.trendData?.points || []
  if (!points.length) return null

  return {
    labels: points.map((point) => point.bucket_start),
    datasets: [
      {
        type: 'bar',
        label: t('admin.dashboard.successRateSuccess'),
        data: points.map((point) => point.success_count),
        backgroundColor: 'rgba(16, 185, 129, 0.8)',
        borderColor: '#059669',
        borderWidth: 1,
        borderRadius: 8,
        stack: 'requests',
        order: 2
      },
      {
        type: 'bar',
        label: t('admin.dashboard.successRateFailed'),
        data: points.map((point) => point.failed_count),
        backgroundColor: 'rgba(244, 63, 94, 0.75)',
        borderColor: '#e11d48',
        borderWidth: 1,
        borderRadius: 8,
        stack: 'requests',
        order: 2
      },
      {
        type: 'line',
        label: t('admin.dashboard.requestSuccessRate'),
        data: points.map((point) => point.success_rate),
        borderColor: '#0f766e',
        backgroundColor: 'rgba(13, 148, 136, 0.18)',
        pointBackgroundColor: '#0f766e',
        pointBorderColor: '#ffffff',
        pointRadius: 3,
        pointHoverRadius: 5,
        tension: 0.35,
        yAxisID: 'yRate',
        order: 1
      }
    ]
  } as any
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: '#6b7280',
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 14,
        font: {
          size: 11
        }
      }
    },
    tooltip: {
      callbacks: {
        label: (context: any) => {
          if (context.dataset.yAxisID === 'yRate') {
            return `${context.dataset.label}: ${formatPercent(context.raw)}`
          }
          return `${context.dataset.label}: ${formatNumber(context.raw)}`
        }
      }
    }
  },
  scales: {
    x: {
      stacked: true,
      grid: {
        color: 'rgba(148, 163, 184, 0.12)'
      },
      ticks: {
        color: '#6b7280',
        maxRotation: 0,
        autoSkip: true,
        font: {
          size: 10
        }
      }
    },
    y: {
      stacked: true,
      beginAtZero: true,
      grid: {
        color: 'rgba(148, 163, 184, 0.14)'
      },
      ticks: {
        color: '#6b7280',
        font: {
          size: 10
        }
      }
    },
    yRate: {
      position: 'right' as const,
      beginAtZero: true,
      min: 0,
      max: 100,
      grid: {
        drawOnChartArea: false
      },
      ticks: {
        color: '#0f766e',
        font: {
          size: 10
        },
        callback: (value: string | number) => `${value}%`
      }
    }
  }
}))

const formatNumber = (value: number | null | undefined): string => {
  if (value === null || value === undefined) return '0'
  return value.toLocaleString()
}

const formatPercent = (value: number | null | undefined): string => {
  if (value === null || value === undefined) return '--'
  return `${value.toFixed(1)}%`
}
</script>
