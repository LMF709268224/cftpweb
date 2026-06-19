import { fileURLToPath } from "node:url"
import { readdir, readFile, stat, writeFile } from "node:fs/promises"
import { extname, join } from "node:path"
import { brotliCompress, constants, gzip } from "node:zlib"
import { promisify } from "node:util"

const gzipAsync = promisify(gzip)
const brotliAsync = promisify(brotliCompress)
const distDir = fileURLToPath(new URL("../dist", import.meta.url))
const compressibleExtensions = new Set([".css", ".html", ".js", ".json", ".svg", ".txt", ".wasm", ".xml"])
const minBytes = 1024

async function walk(dir) {
  const entries = await readdir(dir, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    const fullPath = join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...await walk(fullPath))
      continue
    }
    files.push(fullPath)
  }
  return files
}

async function compressFile(filename) {
  if (!compressibleExtensions.has(extname(filename))) return

  const fileStat = await stat(filename)
  if (fileStat.size < minBytes) return

  const source = await readFile(filename)
  const [gzipped, brotlied] = await Promise.all([
    gzipAsync(source, { level: 9 }),
    brotliAsync(source, {
      params: {
        [constants.BROTLI_PARAM_QUALITY]: 11,
      },
    }),
  ])

  if (gzipped.length < source.length) {
    await writeFile(`${filename}.gz`, gzipped)
  }
  if (brotlied.length < source.length) {
    await writeFile(`${filename}.br`, brotlied)
  }
}

const files = await walk(distDir)
await Promise.all(files.map(compressFile))
console.log("compressed dist assets")
