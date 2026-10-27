import { writeFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const version = process.argv[2] || ''
const identifier = '(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)'
const semver = new RegExp(`^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(?:-${identifier}(?:\\.${identifier})*)?(?:\\+[0-9A-Za-z-]+(?:\\.[0-9A-Za-z-]+)*)?$`)
if (!semver.test(version)) {
  console.error('Usage: make release VERSION=x.y.z (a full semantic version is required)')
  process.exit(1)
}
if (version.includes('+') || version.length > 128) {
  console.error('Docker image tags require a version without build metadata, at most 128 characters.')
  process.exit(1)
}
writeFileSync(new URL('../VERSION', import.meta.url), `${version}\n`)
console.log(`VERSION is now ${version}. Update CHANGELOG.md before committing this release.`)
const result = spawnSync(process.execPath, [fileURLToPath(new URL('./compose.mjs', import.meta.url)), 'build'], { stdio: 'inherit' })
process.exit(result.status ?? 1)
