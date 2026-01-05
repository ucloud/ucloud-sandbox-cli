#!/usr/bin/env -S node --enable-source-maps

import updateNotifier from 'update-notifier'
import * as commander from 'commander'
import { getLastTraceId } from 'e2b'
import * as packageJSON from '../package.json'
import { program } from './commands'
import { commands2md } from './utils/commands2md'

export const pkg = packageJSON

updateNotifier({
  pkg,
  updateCheckInterval: 1000 * 60 * 60 * 8, // 8 hours
}).notify()

const prog = program.version(
  packageJSON.version,
  undefined,
  'display UCloud Sandbox CLI version'
)

if (process.argv.includes('--debug')) {
  process.on('exit', () => {
    const traceId = getLastTraceId()
    console.error(`[DEBUG] Trace ID: ${traceId ?? '(not available)'}`)
  })
}

if (process.env.NODE_ENV === 'development') {
  prog
    .addOption(new commander.Option('-cmd2md').hideHelp())
    .on('option:-cmd2md', () => {
      commands2md(program.commands as any)
      process.exit(0)
    })
}

prog.parse()
