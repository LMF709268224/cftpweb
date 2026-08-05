import { mkdir } from "node:fs/promises"
import { dirname } from "node:path"
import { test as setup } from "@playwright/test"
import { authenticateCandidate, liveAuthStatePath } from "./support/live"

setup.setTimeout(180_000)

setup("authenticate candidate through Casdoor", async ({ page }) => {
  await authenticateCandidate(page)
  await mkdir(dirname(liveAuthStatePath), { recursive: true })
  await page.context().storageState({ path: liveAuthStatePath })
})
