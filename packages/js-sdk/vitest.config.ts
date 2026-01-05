import { defineConfig } from 'vitest/config'
import { config as loadDotenv } from 'dotenv'

const env = loadDotenv()

export default defineConfig({
  test: {
    include: ['tests/**/*.test.ts'],
    exclude: [
      'tests/runtimes/**',
      'tests/integration/**',
      'tests/template/**',
      'tests/connectionConfig.test.ts',
    ],
    isolate: false,
    globals: false,
    testTimeout: 30_000,
    environment: 'node',
    bail: 0,
    deps: {
      interopDefault: true,
    },
    env: {
      ...(process.env as Record<string, string>),
      ...env.parsed,
    },
  },
})

