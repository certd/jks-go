import fs from 'fs'
import path from 'path'
import axios from 'axios'
import { execSync } from 'child_process'

const AtomgitAccessToken = process.env.ATOMGIT_TOKEN
const Owner = process.env.ATOMGIT_OWNER || 'certd'
const Repo = process.env.ATOMGIT_REPO || 'jks-go'
const TargetBranch = process.env.ATOMGIT_TARGET_BRANCH || 'main'
const AssetsDir = process.env.ASSETS_DIR || './release-assets'

function getVersion() {
  if (process.env.VERSION) {
    return process.env.VERSION
  }
  try {
    const tag = execSync('git describe --tags --abbrev=0', { encoding: 'utf-8' }).trim()
    return tag.replace(/^v/, '')
  } catch (e) {
    throw new Error('Cannot determine version. Set VERSION env var or run from a git tag.')
  }
}

function getChangelog(version) {
  try {
    const tags = execSync('git tag --sort=-v:refname', { encoding: 'utf-8' }).trim().split('\n').filter(Boolean)
    const currentTag = `v${version}`
    const idx = tags.indexOf(currentTag)
    if (idx >= 0 && idx < tags.length - 1) {
      const prevTag = tags[idx + 1]
      const log = execSync(`git log --pretty=format:"- %s" ${prevTag}..${currentTag}`, { encoding: 'utf-8' }).trim()
      return log || `Release ${currentTag}`
    }
    const log = execSync(`git log --pretty=format:"- %s" -1`, { encoding: 'utf-8' }).trim()
    return log || `Release ${currentTag}`
  } catch (e) {
    return `Release v${version}`
  }
}

async function createRelease(versionTitle, content) {
  try {
    const response = await axios.request({
      method: 'POST',
      url: `https://api.atomgit.com/api/v5/repos/${Owner}/${Repo}/releases`,
      headers: {
        'Content-Type': 'application/json',
        'PRIVATE-TOKEN': AtomgitAccessToken,
      },
      data: {
        tag_name: `v${versionTitle}`,
        name: `v${versionTitle}`,
        body: content,
        target_commitish: TargetBranch,
      },
    })
    console.log('createRelease success')
    return response.data
  } catch (error) {
    if (error?.response?.status === 409) {
      console.log('Release already exists, skip creation')
      return null
    }
    throw error
  }
}

async function getUploadUrl(versionTitle, fileName) {
  const response = await axios.request({
    method: 'GET',
    url: `https://api.atomgit.com/api/v5/repos/${Owner}/${Repo}/releases/v${versionTitle}/upload_url`,
    headers: {
      'Content-Type': 'application/json',
      'PRIVATE-TOKEN': AtomgitAccessToken,
    },
    params: {
      file_name: fileName,
    },
  })
  console.log(`getUploadUrl success: ${fileName}`)
  return response.data
}

async function uploadFile(url, headers, data) {
  const response = await axios.request({
    method: 'PUT',
    url,
    headers,
    data,
    maxBodyLength: Infinity,
  })
  return response.data
}

async function publishToAtomgit() {
  const versionTitle = getVersion()
  const content = getChangelog(versionTitle)
  console.log(`Version: ${versionTitle}`)

  try {
    const release = await createRelease(versionTitle, content)

    if (!fs.existsSync(AssetsDir)) {
      console.log(`Assets directory ${AssetsDir} not found, skipping asset upload`)
      console.log('publishToAtomgit success (no assets)')
      return
    }

    const files = fs.readdirSync(AssetsDir).filter(f => fs.statSync(path.join(AssetsDir, f)).isFile())
    console.log(`Found ${files.length} files to upload`)

    for (const fileName of files) {
      const filePath = path.join(AssetsDir, fileName)
      const stat = fs.statSync(filePath)
      const fileSizeMB = (stat.size / 1024 / 1024).toFixed(2)
      console.log(`Uploading ${fileName} (${stat.size} bytes / ${fileSizeMB} MB)...`)

      const uploadUrlResponse = await getUploadUrl(versionTitle, fileName)

      if (!uploadUrlResponse?.url) {
        console.error(`  Invalid upload URL response:`, JSON.stringify(uploadUrlResponse).slice(0, 200))
        continue
      }

      console.log(`  Upload URL prefix: ${uploadUrlResponse.url.slice(0, 80)}...`)

      const fileData = fs.createReadStream(filePath)
      uploadUrlResponse.headers['Content-Length'] = stat.size

      await uploadFile(uploadUrlResponse.url, uploadUrlResponse.headers, fileData)
      console.log(`  ${fileName} done`)
    }

    console.log('publishToAtomgit success')
  } catch (error) {
    if (error?.response?.data) {
      console.error('Error response:', JSON.stringify(error.response.data, null, 2))
      throw new Error(`publishToAtomgit error: ${JSON.stringify(error.response.data)}`)
    }
    throw new Error(`publishToAtomgit error: ${error.message}`)
  }
}

publishToAtomgit()
