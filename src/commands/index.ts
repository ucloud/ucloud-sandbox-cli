import * as commander from 'commander'

import { asPrimary } from 'src/utils/format'
import { templateCommand } from './template'
import { sandboxCommand } from './sandbox'
import { authCommand } from './auth'

export const program = new commander.Command()
  .description(
    `Create sandbox templates from Dockerfiles by running ${asPrimary(
      'ucloud-sandbox-cli template build'
    )} then use our SDKs to create sandboxes from these templates.

Visit ${asPrimary(
      'UCloud docs (https://docs.ucloud.cn/modelverse/README)'
    )} to learn how to create sandbox templates and start sandboxes.
`
  )
  .addCommand(authCommand)
  .addCommand(templateCommand)
  .addCommand(sandboxCommand)

function addDebugOption(cmd: commander.Command) {
  cmd.option('--debug', 'print Trace ID for debugging')
  for (const subcommand of cmd.commands) {
    addDebugOption(subcommand)
  }
}

addDebugOption(program)
