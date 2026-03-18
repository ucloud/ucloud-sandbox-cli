import * as commander from 'commander'
import * as fs from 'fs'
import * as path from 'path'
import * as chalk from 'chalk'

import { getUserConfig, USER_CONFIG_PATH } from 'src/user'

const REGIONS = [
  { id: 'cn-wlcb', name: 'cn-wlcb (China - North)' },
  { id: 'us-ca', name: 'us-ca (US - West)' },
] as const

type RegionId = (typeof REGIONS)[number]['id']

export function getRegionDomain(region?: string): string {
  if (!region) return 'sandbox.ucloudai.com'
  return `${region}.sandbox.ucloudai.com`
}

export function getRegionApiUrl(region?: string): string {
  if (!region) return 'https://api.sandbox.ucloudai.com'
  return `https://api.${region}.sandbox.ucloudai.com`
}

function switchRegion(region: string) {
  const valid = REGIONS.find((r) => r.id === region)
  if (!valid) {
    console.error(
      `Unknown region: ${region}\nAvailable regions: ${REGIONS.map((r) => r.id).join(', ')}`
    )
    process.exit(1)
  }

  const userConfig = getUserConfig() || ({} as any)
  const updatedConfig = { ...userConfig, region }
  fs.mkdirSync(path.dirname(USER_CONFIG_PATH), { recursive: true })
  fs.writeFileSync(USER_CONFIG_PATH, JSON.stringify(updatedConfig, null, 2))

  console.log(`Switched to region ${chalk.default.green(valid.name)}`)
  console.log(`  Domain: ${getRegionDomain(region)}`)
  console.log(`  API:    ${getRegionApiUrl(region)}`)
}

export const regionCommand = new commander.Command('region')
  .description('switch region for sandbox services')
  .argument('[region]', 'region to switch to (e.g. cn-wlcb, us-ca)')
  .action(async (region?: string) => {
    if (region) {
      switchRegion(region)
      process.exit(0)
    }

    // Interactive selection
    const userConfig = getUserConfig()
    const currentRegion = userConfig?.region || ''

    const inquirer = await import('inquirer')
    const { selected } = await inquirer.default.prompt([
      {
        name: 'selected',
        type: 'list',
        message: 'Select region:',
        choices: REGIONS.map((r) => ({
          name: r.id === currentRegion
            ? chalk.default.green(`${r.name}  \u2713`)
            : r.name,
          value: r.id,
        })),
        default: currentRegion || REGIONS[0].id,
      },
    ])

    switchRegion(selected)
    process.exit(0)
  })
