import { createGzip } from "node:zlib"
import { createReadStream, createWriteStream, existsSync, readdirSync, statSync } from "node:fs"
import { join } from "node:path"
import { pipeline } from "node:stream/promises"

const root = "dist"
const extensions = new Set([".js", ".css", ".html", ".svg", ".json", ".txt"])
const files = []

function walk(dir) {
  for (const entry of readdirSync(dir)) {
    const fullPath = join(dir, entry)
    const stat = statSync(fullPath)
    if (stat.isDirectory()) {
      walk(fullPath)
      continue
    }

    const dot = entry.lastIndexOf(".")
    const ext = dot >= 0 ? entry.slice(dot) : ""
    if (extensions.has(ext)) {
      files.push(fullPath)
    }
  }
}

if (!existsSync(root)) {
  process.exit(0)
}

walk(root)

for (const file of files) {
  await pipeline(createReadStream(file), createGzip({ level: 9 }), createWriteStream(`${file}.gz`))
}

console.log(`Compressed ${files.length} dist files.`)
