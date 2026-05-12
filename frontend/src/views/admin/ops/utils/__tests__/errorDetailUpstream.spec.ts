import { describe, expect, it } from 'vitest'
import type { OpsErrorDetail } from '@/api/admin/ops'
import { hasUpstreamAttribution, resolveDisplayUpstreamEvents } from '../errorDetailUpstream'

function makeDetail(overrides: Partial<OpsErrorDetail>): OpsErrorDetail {
  return {
    id: 1,
    created_at: '2026-05-12T10:00:00Z',
    phase: 'request',
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
    user_email: 'user@example.com',
    account_id: 101,
    account_name: 'openai-upstream-01',
    group_name: 'default',
    error_body: '{"error":{"message":"openai_error","type":"invalid_request_error"}}',
    user_agent: '',
    request_body: '',
    request_body_truncated: false,
    is_business_limited: false,
    ...overrides
  }
}

describe('errorDetailUpstream', () => {
  it('parses inline upstream_errors events into display rows', () => {
    const detail = makeDetail({
      upstream_errors: JSON.stringify([
        {
          account_id: 101,
          account_name: 'openai-upstream-01',
          platform: 'openai',
          upstream_status_code: 400,
          upstream_request_id: 'req-upstream-1',
          kind: 'http_error',
          message: 'openai_error',
          detail: '{"error":{"message":"openai_error"}}',
          upstream_response_body: '{"error":{"message":"openai_error"}}'
        }
      ])
    })

    const events = resolveDisplayUpstreamEvents(detail)

    expect(events).toHaveLength(1)
    expect(events[0]).toMatchObject({
      key: 'inline-0',
      account_id: 101,
      account_name: 'openai-upstream-01',
      platform: 'openai',
      status_code: 400,
      request_id: 'req-upstream-1',
      kind: 'http_error',
      message: 'openai_error'
    })
    expect(events[0].response_preview).toContain('openai_error')
  })

  it('builds a fallback event from single upstream fields', () => {
    const detail = makeDetail({
      upstream_errors: '',
      upstream_status_code: 422,
      upstream_error_message: 'schema rejected',
      upstream_error_detail: '{"error":"schema rejected"}'
    })

    const events = resolveDisplayUpstreamEvents(detail)

    expect(events).toHaveLength(1)
    expect(events[0]).toMatchObject({
      key: 'fallback',
      account_id: 101,
      account_name: 'openai-upstream-01',
      platform: 'openai',
      status_code: 422,
      request_id: 'gw-rid-1',
      kind: 'http_error',
      message: 'schema rejected'
    })
    expect(events[0].response_preview).toBe('{"error":"schema rejected"}')
  })

  it('reports upstream attribution when account or upstream context exists', () => {
    expect(hasUpstreamAttribution(makeDetail({ account_id: 101 }))).toBe(true)
    expect(hasUpstreamAttribution(makeDetail({ account_id: null, upstream_status_code: 400 }))).toBe(true)
    expect(hasUpstreamAttribution(makeDetail({ account_id: null, upstream_errors: '[{"upstream_status_code":400}]' }))).toBe(true)
    expect(hasUpstreamAttribution(makeDetail({
      account_id: null,
      upstream_status_code: null,
      upstream_error_message: '',
      upstream_error_detail: '',
      upstream_errors: ''
    }))).toBe(false)
  })
})
