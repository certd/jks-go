import { createInterface } from 'node:readline'
import { spawn, execSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import fs from 'node:fs'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

function ask(query) {
  return new Promise(resolve => {
    const rl = createInterface({ input: process.stdin, output: process.stdout })
    rl.question(query, answer => {
      rl.close()
      resolve(answer.trim())
    })
  })
}

function exec(cmd) {
  return execSync(cmd, { encoding: 'utf-8', stdio: ['pipe', 'pipe', 'pipe'] }).trim()
}

function getRepoSlug() {
  try {
    const url = exec('git remote get-url origin')
    const m = url.match(/github\.com[/:]([^/]+)\/([^/.]+?)(?:\.git)?\s*$/)
    if (m) return `${m[1]}/${m[2]}`
  } catch (e) { /* ignore */ }
  return process.env.GITHUB_REPOSITORY || 'certd/jks-go'
}

async function downloadFromGitHub(version) {
  const repo = getRepoSlug()
  const tag = `v${version}`
  const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN
  const proxy = process.env.HTTPS_PROXY || process.env.HTTP_PROXY || 'http://127.0.0.1:10811'
  if (!token) {
    console.error('GITHUB_TOKEN or GH_TOKEN environment variable is required')
    process.exit(1)
  }

  console.log(`Fetching release ${tag} from ${repo} ...`)

  const releaseJson = exec(`curl -s -x "${proxy}" -H "Authorization: token ${token}" -H "Accept: application/vnd.github+json" "https://api.github.com/repos/${repo}/releases/tags/${tag}"`)

  let release
  try {
    release = JSON.parse(releaseJson)
  } catch (e) {
    console.error('Failed to parse GitHub API response')
    console.error(releaseJson.slice(0, 500))
    process.exit(1)
  }

  if (!release.id) {
    console.error(`Release ${tag} not found`)
    process.exit(1)
  }

  const distDir = path.resolve(__dirname, '..', 'dist')
  fs.rmSync(distDir, { recursive: true, force: true })
  fs.mkdirSync(distDir, { recursive: true })

  console.log(`Downloading ${release.assets.length} assets...`)

  for (const asset of release.assets) {
    const name = asset.name
    process.stdout.write(`  ${name} ... `)
    exec(`curl -sL -x "${proxy}" -H "Authorization: token ${token}" -H "Accept: application/octet-stream" -o "${distDir}/${name}" "${asset.url}"`)
    const filePath = path.join(distDir, name)
    if (fs.existsSync(filePath)) {
      const size = fs.statSync(filePath).size
      console.log(`${(size / 1024).toFixed(1)} KB`)
    } else {
      console.log('FAILED')
    }
  }

  return distDir
}

async function main() {
  let version = await ask('Version (without v prefix): ')
  if (!version) {
    console.error('Version is required')
    process.exit(1)
  }

  const distDir = await downloadFromGitHub(version)

  const files = fs.readdirSync(distDir).filter(f => fs.statSync(path.join(distDir, f)).isFile())
  if (files.length === 0) {
    console.error(`No files downloaded`)
    process.exit(1)
  }

  const confirm = await ask('\nPublish to atomgit? (y/N): ')
  if (confirm.toLowerCase() !== 'y') {
    console.log('Aborted')
    process.exit(0)
  }

  console.log('\nPublishing...')
  const env = {
    ...process.env,
    VERSION: version,
    ASSETS_DIR: distDir,
  }

  const proc = spawn('node', [path.join(__dirname, 'publish-atomgit.js')], {
    env,
    stdio: 'inherit',
    cwd: __dirname,
  })

  proc.on('exit', code => {
    process.exit(code ?? 1)
  })
}

main()
