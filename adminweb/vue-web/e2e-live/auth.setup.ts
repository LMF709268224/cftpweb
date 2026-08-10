import { mkdir } from "node:fs/promises"
import { dirname } from "node:path"
import { test as setup } from "@playwright/test"
import { authenticateAdmin, liveAuthStatePath } from "./support/live"

setup.setTimeout(180_000)

setup("authenticate admin through Casdoor", async ({ page }) => {
  await authenticateAdmin(page)
  await mkdir(dirname(liveAuthStatePath), { recursive: true })
  await page.context().storageState({ path: liveAuthStatePath })
})
