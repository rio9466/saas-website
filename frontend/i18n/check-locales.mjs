// Verifies that every locale file in frontend/i18n/locales exposes the same set
// of translation keys. Run through `pnpm lint`.
import { readFileSync, readdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const localesDir = join(dirname(fileURLToPath(import.meta.url)), 'locales')

/** @param {Record<string, unknown>} value @param {string} prefix */
function flatten(value, prefix = '') {
  const keys = []
  for (const [key, child] of Object.entries(value)) {
    const path = prefix ? `${prefix}.${key}` : key
    if (child && typeof child === 'object' && !Array.isArray(child)) {
      keys.push(...flatten(/** @type {Record<string, unknown>} */ (child), path))
    } else {
      keys.push(path)
    }
  }
  return keys
}

const files = readdirSync(localesDir).filter(file => file.endsWith('.json')).sort()

if (files.length === 0) {
  console.error('[i18n] no locale files found in', localesDir)
  process.exit(1)
}

const reference = files[0]
const referenceKeys = new Set(flatten(JSON.parse(readFileSync(join(localesDir, reference), 'utf8'))))
let failed = false

for (const file of files.slice(1)) {
  const keys = new Set(flatten(JSON.parse(readFileSync(join(localesDir, file), 'utf8'))))
  const missing = [...referenceKeys].filter(key => !keys.has(key))
  const extra = [...keys].filter(key => !referenceKeys.has(key))
  if (missing.length || extra.length) {
    failed = true
    console.error(`[i18n] key mismatch between ${reference} and ${file}`)
    if (missing.length) console.error(`  missing in ${file}: ${missing.join(', ')}`)
    if (extra.length) console.error(`  only in ${file}: ${extra.join(', ')}`)
  }
}

if (failed) {
  process.exit(1)
}

console.log(`[i18n] ${files.length} locale files share the same ${referenceKeys.size} keys (${files.join(', ')})`)
