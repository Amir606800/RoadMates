import { readFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const root = fileURLToPath(new URL('../', import.meta.url))
const appVersion = readFileSync(new URL('../VERSION', import.meta.url), 'utf8').trim()
const git = spawnSync('git', ['rev-parse', '--short=7', 'HEAD'], { cwd: root, encoding: 'utf8' })
const env = { ...process.env, APP_VERSION: appVersion, COMMIT: process.env.COMMIT || (git.status === 0 ? git.stdout.trim() : 'unknown'), BUILD_TIME: process.env.BUILD_TIME || new Date().toISOString() }
const result = spawnSync('docker', ['compose', ...process.argv.slice(2)], { cwd: root, env, stdio: 'inherit' })
if (result.error) console.error(result.error.message)
process.exit(result.status ?? 1)
