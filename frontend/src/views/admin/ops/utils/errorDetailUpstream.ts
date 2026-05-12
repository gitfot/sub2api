import type { OpsErrorDetail } from '@/api/admin/ops'

export type DisplayUpstreamEvent = {
  key: string
  account_id?: number | null
  account_name?: string
  platform?: string
  status_code?: number | null
  request_id?: string
  kind?: string
  message?: string
  detail?: string
  response_preview?: string
}

type RawUpstreamEvent = {
  account_id?: number
  account_name?: string
  platform?: string
  upstream_status_code?: number
  upstream_request_id?: string
  kind?: string
  message?: string
  detail?: string
  upstream_response_body?: string
}

function clean(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function parseRawEvents(raw: string | undefined): RawUpstreamEvent[] {
  const text = clean(raw)
  if (!text || text === '[]' || text === '{}' || text.toLowerCase() === 'null') return []
  try {
    const parsed = JSON.parse(text)
    return Array.isArray(parsed) ? parsed.filter((item): item is RawUpstreamEvent => !!item && typeof item === 'object') : []
  } catch {
    return []
  }
}

export function resolveDisplayUpstreamEvents(detail: OpsErrorDetail | null | undefined): DisplayUpstreamEvent[] {
  if (!detail) return []

  const inline = parseRawEvents(detail.upstream_errors).map((ev, index) => ({
    key: `inline-${index}`,
    account_id: ev.account_id ?? null,
    account_name: clean(ev.account_name),
    platform: clean(ev.platform),
    status_code: ev.upstream_status_code ?? null,
    request_id: clean(ev.upstream_request_id),
    kind: clean(ev.kind),
    message: clean(ev.message),
    detail: clean(ev.detail),
    response_preview: clean(ev.upstream_response_body) || clean(ev.detail)
  }))

  if (inline.length > 0) return inline

  const status = detail.upstream_status_code ?? null
  const message = clean(detail.upstream_error_message)
  const detailPayload = clean(detail.upstream_error_detail)
  const hasFallback = detail.account_id != null || status != null || message || detailPayload
  if (!hasFallback) return []

  return [{
    key: 'fallback',
    account_id: detail.account_id ?? null,
    account_name: clean(detail.account_name),
    platform: clean(detail.platform),
    status_code: status,
    request_id: clean(detail.request_id) || clean(detail.client_request_id),
    kind: 'http_error',
    message,
    detail: detailPayload,
    response_preview: detailPayload || message
  }]
}

export function hasUpstreamAttribution(detail: OpsErrorDetail | null | undefined): boolean {
  return resolveDisplayUpstreamEvents(detail).length > 0
}
