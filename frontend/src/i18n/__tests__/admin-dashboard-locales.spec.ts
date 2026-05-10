import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const requiredAdminDashboardKeys = [
  'requestSuccessRate',
  'accountFilter',
  'allAccounts',
  'todaySuccessRate',
  'historySuccessRate',
  'successRateTrend',
  'successRateRequests',
  'successRateSuccess',
  'successRateFailed',
  'successRateGranularity10m',
  'successRateGranularity1h',
  'successRateGranularity1d'
] as const

describe('admin dashboard locales', () => {
  it.each([
    ['en', en],
    ['zh', zh]
  ])('defines account success-rate labels under admin.dashboard for %s', (_locale, messages) => {
    const adminDashboard = messages.admin?.dashboard

    expect(adminDashboard).toBeTruthy()

    for (const key of requiredAdminDashboardKeys) {
      expect(adminDashboard[key]).toBeTypeOf('string')
      expect(adminDashboard[key].length).toBeGreaterThan(0)
    }
  })
})
