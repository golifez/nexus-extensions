import { readdir } from 'node:fs/promises'
import { readFile } from 'node:fs/promises'
import path from 'node:path'

const root = path.resolve('extensions')
const idPattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/
const semverPattern = /^v?[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$/
const errors = []

let entries = []
try {
  entries = await readdir(root, { withFileTypes: true })
} catch (error) {
  console.error(`cannot read extensions/: ${error.message}`)
  process.exit(1)
}

for (const entry of entries) {
  if (!entry.isDirectory()) continue
  const id = entry.name
  const dir = path.join(root, id)
  const manifestPath = path.join(dir, 'manifest.json')
  let manifest
  try {
    manifest = JSON.parse(await readFile(manifestPath, 'utf8'))
  } catch (error) {
    errors.push(`${id}: manifest.json is missing or invalid (${error.message})`)
    continue
  }
  if (!idPattern.test(id) || manifest.id !== id) errors.push(`${id}: directory name and manifest.id must match kebab-case`) 
  for (const field of ['name', 'description', 'apiVersion', 'sourceRef']) {
    if (typeof manifest[field] !== 'string' || !manifest[field].trim()) errors.push(`${id}: ${field} is required`)
  }
  if (!semverPattern.test(manifest.version || '')) errors.push(`${id}: version must be SemVer`)
  if (!manifest.entry?.binary) errors.push(`${id}: entry.binary is required`)
  if (!manifest.ui?.entryPath?.startsWith('/')) errors.push(`${id}: ui.entryPath must start with /`)
  if (!Array.isArray(manifest.permissions)) errors.push(`${id}: permissions must be an array`)
  for (const file of ['README.md']) {
    try { await readFile(path.join(dir, file)) } catch { errors.push(`${id}: ${file} is required`) }
  }
}

if (errors.length) {
  console.error(errors.map((error) => `- ${error}`).join('\n'))
  process.exit(1)
}
console.log(`validated ${entries.filter((entry) => entry.isDirectory()).length} extension manifest(s)`)

