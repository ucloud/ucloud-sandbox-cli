import * as commander from 'commander'
import * as fs from 'fs'
import * as path from 'path'
import * as open from 'open'

import {
  getUserConfig,
  USER_CONFIG_PATH,
  UserConfig,
} from 'src/user'
import { asFormattedConfig, asFormattedError, asPrimary } from 'src/utils/format'

const API_KEY_URL = 'https://console.ucloud.cn/modelverse/experience/api-keys'

export const loginCommand = new commander.Command('login')
  .description('log in to CLI')
  .action(async () => {
    let userConfig: UserConfig | null = null

    try {
      userConfig = getUserConfig()
    } catch (err) {
      console.error(asFormattedError('Failed to read user config', err))
    }

    if (userConfig) {
      console.log(
        `\nAlready logged in. ${asFormattedConfig(
          userConfig
        )}.\n\nIf you want to log in as a different user, log out first by running 'ucloud-sandbox-cli auth logout'.\nTo change the team, run 'ucloud-sandbox-cli auth configure'.\n`
      )
      return
    }

    console.log(`Log in to CLI.\n`)
    console.log(`Get your API Key from: ${asPrimary(API_KEY_URL)}\n`)

    // Try to open browser to API key page (may fail on headless servers)
    try {
      const childProcess = await open.default(API_KEY_URL, { wait: false })
      childProcess.on('error', () => {
        // Silently ignore if browser can't be opened
      })
    } catch (err) {
      // Silently ignore if browser can't be opened
    }

    const inquirer = await import('inquirer')

    const { apiKey } = await inquirer.default.prompt([
      {
        type: 'password',
        name: 'apiKey',
        message: 'Enter your API Key:',
        mask: '*',
        validate: (input: string) => {
          if (!input.trim()) {
            return 'API Key cannot be empty'
          }
          return true
        },
      },
    ])

    userConfig = {
      email: 'api-key-user',
      accessToken: apiKey,
      teamName: 'default',
      teamId: 'default',
      teamApiKey: apiKey,
    }

    fs.mkdirSync(path.dirname(USER_CONFIG_PATH), { recursive: true })
    fs.writeFileSync(USER_CONFIG_PATH, JSON.stringify(userConfig, null, 2))

    console.log(`\nLogged in successfully.`)
    process.exit(0)
  })
