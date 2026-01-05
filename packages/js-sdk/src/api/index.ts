import createClient, { FetchResponse } from 'openapi-fetch'

import type { components, paths } from './schema.gen'
import { defaultHeaders } from './metadata'
import { ConnectionConfig } from '../connectionConfig'
import { AuthenticationError, RateLimitError, SandboxError } from '../errors'
import { createApiLogger } from '../logs'
import {
  appendTraceIdToMessage,
  createTraceIdMiddleware,
  getTraceIdFromResponse,
} from '../trace'

export function handleApiError(
  response: FetchResponse<any, any, any>,
  errorClass: new (
    message: string,
    stackTrace?: string
  ) => Error = SandboxError,
  stackTrace?: string
): Error | undefined {
  if (!response.error) {
    return
  }

  const traceId = getTraceIdFromResponse(response.response)

  if (response.response.status === 401) {
    const message = 'Unauthorized, please check your credentials.'
    const content = response.error?.message ?? response.error

    if (content) {
      const err = new AuthenticationError(
        appendTraceIdToMessage(`${message} - ${content}`, traceId)
      )
      ;(err as any).traceId = traceId
      return err
    }
    const err = new AuthenticationError(appendTraceIdToMessage(message, traceId))
    ;(err as any).traceId = traceId
    return err
  }

  if (response.response.status === 429) {
    const message = 'Rate limit exceeded, please try again later'
    const content = response.error?.message ?? response.error

    if (content) {
      const err = new RateLimitError(
        appendTraceIdToMessage(`${message} - ${content}`, traceId)
      )
      ;(err as any).traceId = traceId
      return err
    }
    const err = new RateLimitError(appendTraceIdToMessage(message, traceId))
    ;(err as any).traceId = traceId
    return err
  }

  const message = response.error?.message ?? response.error
  const err = new errorClass(
    appendTraceIdToMessage(`${response.response.status}: ${message}`, traceId),
    stackTrace
  )
  ;(err as any).traceId = traceId
  return err
}

/**
 * Client for interacting with the UCloud Sandbox API.
 */
class ApiClient {
  readonly api: ReturnType<typeof createClient<paths>>

  constructor(
    config: ConnectionConfig,
    opts: {
      requireAccessToken?: boolean
      requireApiKey?: boolean
    } = { requireAccessToken: false, requireApiKey: false }
  ) {
    if (opts?.requireApiKey && !config.apiKey) {
      throw new AuthenticationError(
        'API key is required, please visit https://console.ucloud.cn/modelverse/experience/api-keys to get your API key. ' +
        'You can either set the environment variable `E2B_API_KEY` ' +
        "or you can pass it directly to the sandbox like Sandbox.create({ apiKey: '...' })"
      )
    }

    if (opts?.requireAccessToken && !config.accessToken) {
      throw new AuthenticationError(
        'Access token is required, please visit https://console.ucloud.cn/modelverse/experience/api-keys to get your access token. ' +
        'You can set the environment variable `E2B_ACCESS_TOKEN` or pass the `accessToken` in options.'
      )
    }

    this.api = createClient<paths>({
      baseUrl: config.apiUrl,
      // keepalive: true, // TODO: Return keepalive
      headers: {
        ...defaultHeaders,
        ...(config.apiKey && { 'X-API-Key': config.apiKey }),
        ...config.headers,
      },
      querySerializer: {
        array: {
          style: 'form',
          explode: false,
        },
      },
    })

    this.api.use(createTraceIdMiddleware())

    if (config.logger) {
      this.api.use(createApiLogger(config.logger))
    }
  }
}

export type { components, paths }
export { ApiClient }
