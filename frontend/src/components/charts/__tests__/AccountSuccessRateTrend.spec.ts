import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountSuccessRateTrend from '../AccountSuccessRateTrend.vue'

const messages: Record<string, string> = {
  'admin.dashboard.successRateTrend': 'Account Success Rate Trend',
  'admin.dashboard.accountFilter': 'Account',
  'admin.dashboard.allAccounts': 'All Accounts',
  'admin.dashboard.successRateGranularity10m': '10m',
  'admin.dashboard.successRateGranularity1h': '1h',
  'admin.dashboard.successRateGranularity1d': '1d',
  'admin.dashboard.successRateRequests': 'Requests',
  'admin.dashboard.requestSuccessRate': 'Request Success Rate',
  'admin.dashboard.todaySuccessRate': 'Today Success Rate',
  'admin.dashboard.historySuccessRate': 'History Success Rate',
  'admin.dashboard.successRateSuccess': 'Success',
  'admin.dashboard.successRateFailed': 'Failed',
  'admin.dashboard.noDataAvailable': 'No data available',
  'common.refresh': 'Refresh',
  'common.loading': 'Loading'
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key
    })
  }
})

vi.mock('vue-chartjs', () => ({
  Bar: {
    props: ['data'],
    template: '<div class="chart-data">{{ JSON.stringify(data) }}</div>'
  }
}))

describe('AccountSuccessRateTrend', () => {
  it('renders account filter options', () => {
    const wrapper = mount(AccountSuccessRateTrend, {
      props: {
        loading: false,
        granularity: '1h',
        selectedAccountId: 0,
        accountOptions: [
          { value: 0, label: 'All Accounts' },
          { value: 42, label: 'claude-01' }
        ],
        trendData: {
          bucket: '1h',
          computed_at: '2026-05-11T10:00:00Z',
          stale: false,
          partial: false,
          points: []
        }
      },
      global: {
        stubs: {
          LoadingSpinner: true,
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue'],
            template: '<div class="select-stub">{{ options.map((option) => option.label).join(",") }}</div>'
          },
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('Account')
    expect(wrapper.text()).toContain('All Accounts')
    expect(wrapper.text()).toContain('claude-01')
  })

  it('renders empty state when no points are returned', () => {
    const wrapper = mount(AccountSuccessRateTrend, {
      props: {
        loading: false,
        granularity: '1h',
        selectedAccountId: 0,
        accountOptions: [{ value: 0, label: 'All Accounts' }],
        trendData: {
          bucket: '1h',
          computed_at: '2026-05-11T10:00:00Z',
          stale: false,
          partial: false,
          points: []
        }
      },
      global: {
        stubs: {
          LoadingSpinner: true,
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue'],
            template: '<div class="select-stub">{{ options.map((option) => option.label).join(",") }}</div>'
          },
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('No data available')
  })
})
