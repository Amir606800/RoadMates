import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const root = fileURLToPath(new URL('../', import.meta.url))
function run(args) {
  const result = spawnSync('docker', args, { cwd: root, stdio: 'inherit' })
  if (result.error) console.error(result.error.message)
  return result.status ?? 1
}
const frontend = run(['build', '--target', 'test', '-t', 'roadmates-frontend-test', 'frontend'])
if (frontend !== 0) process.exit(frontend)
const compose = ['compose', '-p', 'roadmates-tests', '-f', 'docker-compose.test.yml']
let status = 1
try {
  status = run([...compose, 'up', '--build', '--abort-on-container-exit', '--exit-code-from', 'backend-test'])
} finally {
  const cleanup = run([...compose, 'down', '--volumes', '--remove-orphans'])
  if (status === 0) status = cleanup
}
process.exit(status)
