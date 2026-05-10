import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountSuccessRateCell from '../AccountSuccessRateCell.vue'

describe('AccountSuccessRateCell', () => {
  it('renders -- when request_count is zero', () => {
    const wrapper = mount(AccountSuccessRateCell, {
      props: {
        stats: {
          success_count: 0,
          failed_count: 0,
          request_count: 0,
          success_rate: null
        }
      }
    })

    expect(wrapper.text()).toContain('--')
  })

  it('renders success rate and request counts when stats are populated', () => {
    const wrapper = mount(AccountSuccessRateCell, {
      props: {
        stats: {
          success_count: 19,
          failed_count: 1,
          request_count: 20,
          success_rate: 95
        }
      }
    })

    expect(wrapper.text()).toContain('95.00%')
    expect(wrapper.text()).toContain('19 / 1')
  })

  it('renders placeholder when the batch request degrades', () => {
    const wrapper = mount(AccountSuccessRateCell, {
      props: {
        loading: false,
        error: 'Failed',
        stats: null
      }
    })

    expect(wrapper.text()).toContain('--')
  })
})
