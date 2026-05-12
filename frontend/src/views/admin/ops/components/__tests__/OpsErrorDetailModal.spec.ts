import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OpsErrorDetailModal from '../OpsErrorDetailModal.vue'

const {
  mockGetRequestErrorDetail,
  mockListRequestErrorUpstreamErrors
} = vi.hoisted(() => ({
  mockGetRequestErrorDetail: vi.fn(),
  mockListRequestErrorUpstreamErrors: vi.fn()
}))

vi.mock('@/api/admin/ops', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/admin/ops')>()
  return {
    ...actual,
    opsAPI: {
      ...actual.opsAPI,
      getRequestErrorDetail: (...args: any[]) => mockGetRequestErrorDetail(...args),
      listRequestErrorUpstreamErrors: (...args: any[]) => mockListRequestErrorUpstreamErrors(...args)
    }
  }
})

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

describe('OpsErrorDetailModal', () => {
  beforeEach(() => {
    mockGetRequestErrorDetail.mockReset()
    mockListRequestErrorUpstreamErrors.mockReset()
    mockListRequestErrorUpstreamErrors.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100 })
  })

  it('shows user and upstream account for a request error with inline upstream events', async () => {
    mockGetRequestErrorDetail.mockResolvedValue({
      id: 7,
      created_at: '2026-05-12T10:00:00Z',
      phase: 'upstream',
      type: 'invalid_request_error',
      error_owner: 'provider',
      error_source: 'upstream_http',
      severity: 'P3',
      status_code: 400,
      platform: 'openai',
      model: 'gpt-4o-mini',
      is_retryable: false,
      retry_count: 0,
      resolved: false,
      client_request_id: 'client-rid-1',
      request_id: 'gw-rid-1',
      message: 'openai_error',
      user_id: 55,
      user_email: 'user@example.com',
      account_id: 101,
      account_name: 'openai-upstream-01',
      group_name: 'default',
      request_path: '/v1/chat/completions',
      stream: false,
      inbound_endpoint: '/v1/chat/completions',
      upstream_endpoint: '/v1/chat/completions',
      requested_model: 'gpt-4o-mini',
      upstream_model: 'gpt-4o-mini',
      error_body: '{"error":{"message":"openai_error","type":"invalid_request_error"}}',
      user_agent: 'test-client',
      upstream_status_code: 400,
      upstream_error_message: 'openai_error',
      upstream_error_detail: '{"error":{"message":"openai_error"}}',
      upstream_errors: JSON.stringify([
        {
          account_id: 101,
          account_name: 'openai-upstream-01',
          upstream_status_code: 400,
          upstream_request_id: 'req-upstream-1',
          kind: 'http_error',
          message: 'openai_error',
          detail: '{"error":{"message":"openai_error"}}'
        }
      ]),
      request_body: '',
      request_body_truncated: false,
      is_business_limited: false
    })

    const wrapper = mount(OpsErrorDetailModal, {
      props: { show: true, errorId: 7, errorType: 'request' },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /></div>' },
          Icon: { template: '<span />' }
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('user@example.com')
    expect(wrapper.text()).toContain('openai-upstream-01')
    expect(wrapper.text()).toContain('101')
    expect(wrapper.text()).toContain('req-upstream-1')
    expect(wrapper.text()).toContain('openai_error')
  })
})
