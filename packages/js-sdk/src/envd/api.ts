import createClient from 'openapi-fetch'

import type { components, paths } from './schema.gen'
import { ConnectionConfig } from '../connectionConfig'
import { createApiLogger } from '../logs'
import {
  SandboxError,
  InvalidArgumentError,
  NotFoundError,
  NotEnoughSpaceError,
  formatSandboxTimeoutError,
  AuthenticationError,
} from '../errors'
import {
  appendTraceIdToMessage,
  createTraceIdMiddleware,
  getTraceIdFromResponse,
} from '../trace'
import { StartResponse, ConnectResponse } from './process/process_pb'
import { Code, ConnectError } from '@connectrpc/connect'
import { WatchDirResponse } from './filesystem/filesystem_pb'

type ApiError = { message?: string } | string

export async function handleEnvdApiError(res: {
  error?: ApiError
  response: Response
}) {
  if (!res.error) {
    return
  }

  const traceId = getTraceIdFromResponse(res.response)

  const message: string =
    typeof res.error == 'string'
      ? res.error
      : res.error?.message || (await res.response.text())

  let err: Error
  switch (res.response.status) {
    case 400:
      err = new InvalidArgumentError(message)
      break
    case 401:
      err = new AuthenticationError(message)
      break
    case 404:
      err = new NotFoundError(message)
      break
    case 429:
      err = new SandboxError(
        appendTraceIdToMessage(
          `${res.response.status}: ${message}: The requests are being rate limited.`,
          traceId
        )
      )
      break
    case 502:
      err = formatSandboxTimeoutError(message)
      break
    case 507:
      err = new NotEnoughSpaceError(message)
      break
    default:
      err = new SandboxError(`${res.response.status}: ${message}`)
      break
  }

  err.message = appendTraceIdToMessage(err.message, traceId)
  ;(err as any).traceId = traceId
  return err
}

export async function handleProcessStartEvent(
  events: AsyncIterable<StartResponse | ConnectResponse>
) {
  let startEvent: StartResponse | ConnectResponse

  try {
    startEvent = (await events[Symbol.asyncIterator]().next()).value
  } catch (err) {
    if (err instanceof ConnectError) {
      if (err.code === Code.Unavailable) {
        throw new NotFoundError('Sandbox is probably not running anymore')
      }
    }

    throw err
  }
  if (startEvent.event?.event.case !== 'start') {
    throw new Error('Expected start event')
  }

  return startEvent.event.event.value.pid
}

export async function handleWatchDirStartEvent(
  events: AsyncIterable<WatchDirResponse>
) {
  let startEvent: WatchDirResponse

  try {
    startEvent = (await events[Symbol.asyncIterator]().next()).value
  } catch (err) {
    if (err instanceof ConnectError) {
      if (err.code === Code.Unavailable) {
        throw new NotFoundError('Sandbox is probably not running anymore')
      }
    }

    throw err
  }
  if (startEvent.event?.case !== 'start') {
    throw new Error('Expected start event')
  }

  return startEvent.event.value
}

class EnvdApiClient {
  readonly api: ReturnType<typeof createClient<paths>>
  readonly version: string

  constructor(
    config: Pick<ConnectionConfig, 'apiUrl' | 'logger' | 'accessToken'> & {
      fetch?: (request: Request) => ReturnType<typeof fetch>
      headers?: Record<string, string>
    },
    metadata: {
      version: string
    }
  ) {
    this.api = createClient({
      baseUrl: config.apiUrl,
      fetch: config?.fetch,
      headers: config?.headers,
      // keepalive: true, // TODO: Return keepalive
    })
    this.version = metadata.version

    this.api.use(createTraceIdMiddleware())

    if (config.logger) {
      this.api.use(createApiLogger(config.logger))
    }
  }
}

export type { components, paths }
export { EnvdApiClient }
