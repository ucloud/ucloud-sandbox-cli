import type { Middleware } from 'openapi-fetch'

export const TRACE_ID_HEADER = 'x-trace-id'

let lastTraceId: string | undefined

export function getLastTraceId(): string | undefined {
  return lastTraceId
}

export function setLastTraceId(traceId: string | null | undefined) {
  if (typeof traceId !== 'string') {
    return
  }

  const trimmed = traceId.trim()
  if (!trimmed) {
    return
  }

  lastTraceId = trimmed
}

export function getTraceIdFromResponse(
  response: Response | undefined | null
): string | undefined {
  return response?.headers?.get(TRACE_ID_HEADER) ?? undefined
}

export function appendTraceIdToMessage(
  message: string,
  traceId: string | undefined
): string {
  if (!traceId) {
    return message
  }

  if (message.toLowerCase().includes('x-trace-id:')) {
    return message
  }

  return `${message}\nX-Trace-ID: ${traceId}`
}

export function createTraceIdMiddleware(): Middleware {
  return {
    async onResponse({ response }) {
      setLastTraceId(getTraceIdFromResponse(response))
      return response
    },
  }
}
