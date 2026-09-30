// Belgeler arasındaki göreli bağlantıları doğrula. Bölümler ayrılırken taşınmış içeriğe işaret eden bozuk bağlantılar bırakmak en kolay hatadır,
// ve vitepress build bozuk bağlantılar nedeniyle başarısız olmaz.
import { readFileSync, readdirSync, statSync, existsSync } from 'node:fs'
import { join, relative, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    if (['node_modules', 'homepage', 'shared', 'scripts', 'deploy', 'static-site', 'releases'].includes(name) || name.startsWith('.')) continue
    const path = join(dir, name)
    if (statSync(path).isDirectory()) walk(path, out)
    else if (name.endsWith('.md')) out.push(path)
  }
  return out
}

const files = walk(root)
const failures = []
let checked = 0

for (const file of files) {
  const lines = readFileSync(file, 'utf-8').split('\n')
  lines.forEach((line, index) => {
    for (const match of line.matchAll(/\[[^\]]*\]\(([^)\s]+)\)/g)) {
      const target = match[1]
      if (/^(https?:|#|mailto:)/.test(target)) continue
      const [path] = target.split('#')
      if (!path || !path.endsWith('.md')) continue
      checked++
      if (!existsSync(resolve(dirname(file), path))) {
        failures.push({ file: relative(root, file), line: index + 1, target })
      }
    }
  })
}

console.log(`${files.length} belgedeki ${checked} dahili bağlantı kontrol edildi, ${failures.length} geçersiz`)
for (const failure of failures) {
  console.log(`  ${failure.file}:${failure.line} -> ${failure.target}`)
}
process.exit(failures.length === 0 ? 0 : 1)
