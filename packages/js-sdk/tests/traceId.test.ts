import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

import { handleApiError } from '../src/api'
import { handleEnvdApiError } from '../src/envd/api'
import { Sandbox, getLastTraceId } from '../src'
import { SandboxError } from '../src/errors'

const server = setupServer(
  http.get(
    'https://api.sandbox.ucloudai.com/sandboxes/:sandboxID',
    async () => {
      return HttpResponse.json(
        { code: 404, message: 'not found' },
        {
          status: 404,
          headers: {
            'X-Trace-ID': 'trace-404',
          },
        }
      )
    }
  )
)

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('X-Trace-ID propagation', () => {
  it('appends X-Trace-ID to handleApiError messages', () => {
    const response = new Response(
      JSON.stringify({ code: 500, message: 'internal error' }),
      {
        status: 500,
        headers: {
          'X-Trace-ID': 'trace-500',
        },
      }
    )

    const err = handleApiError({
      error: { message: 'internal error' },
      response,
    } as any)

    expect(err).toBeInstanceOf(SandboxError)
    expect(err?.message).toContain('X-Trace-ID: trace-500')
    expect((err as any)?.traceId).toBe('trace-500')
  })

  it('appends X-Trace-ID to handleEnvdApiError messages', async () => {
    const response = new Response('unauthorized', {
      status: 401,
      headers: {
        'X-Trace-ID': 'trace-envd-401',
      },
    })

    const err = await handleEnvdApiError({
      error: 'unauthorized',
      response,
    })

    expect(err?.message).toContain('X-Trace-ID: trace-envd-401')
    expect((err as any)?.traceId).toBe('trace-envd-401')
  })

  it('includes X-Trace-ID in NotFoundError messages and updates last trace id', async () => {
    await expect(Sandbox.getInfo('missing-sandbox')).rejects.toMatchObject({
      message: expect.stringContaining('X-Trace-ID: trace-404'),
    })

    expect(getLastTraceId()).toBe('trace-404')
  })
})

