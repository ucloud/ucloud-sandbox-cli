import * as os from 'os'
import * as path from 'path'
import * as fs from 'fs'

/**
 * User configuration stored in ~/.e2b/config.json
 */
export interface UserConfig {
  email: string
  accessToken: string
  teamName: string
  teamId: string
  teamApiKey: string
  dockerProxySet?: boolean
}

export const USER_CONFIG_PATH = path.join(os.homedir(), '.ucloud-sandbox-cli', 'config.json')
export const DOCS_BASE =
  process.env.UCLOUD_SANDBOX_DOCS_BASE ||
  `https://${process.env.UCLOUD_SANDBOX_DOMAIN || 'sandbox.ucloudai.com'}/docs`

export function getUserConfig(): UserConfig | null {
  if (!fs.existsSync(USER_CONFIG_PATH)) return null
  try {
    const content = fs.readFileSync(USER_CONFIG_PATH, 'utf8')
    if (!content || content.trim() === '') return null
    return JSON.parse(content)
  } catch {
    // Return null if config file is empty or contains invalid JSON
    return null
  }
}
