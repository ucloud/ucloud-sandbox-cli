import status from 'statuses'

/**
 * Thrown when a request to Sandbox API occurs.
 */
export class SandboxRequestError extends Error {
  constructor(message: any) {
    super(message)
    this.name = 'SandboxRequestError'
  }
}

export function handleSandboxRequestError<T>(
  res: {
    data?: T | null | undefined
    error?: { code: number; message: string }
  },
  errMsg?: string
): asserts res is { data: T; error?: undefined } {
  if (!res.error) {
    return
  }

  let message: string
  const code = res.error?.code ?? 0
  switch (code) {
    case 400:
      message = 'bad request'
      break
    case 401:
      message = 'unauthorized'
      break
    case 403:
      message = 'forbidden'
      break
    case 404:
      message = 'not found'
      break
    case 500:
      message = 'internal server error'
      break
    default:
      message = status(code) || 'unknown error'
      break
  }

  throw new SandboxRequestError(
    `${errMsg && `${errMsg}: `}[${code}] ${message && `${message}: `}${
      res.error?.message ?? 'no message'
    }`
  )
}
