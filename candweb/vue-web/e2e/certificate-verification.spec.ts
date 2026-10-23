import { expect, test } from "@playwright/test"

const samplePdf = Buffer.from(`%PDF-1.4\n${"x".repeat(40)}`)

test("shows an invalid result and keeps technical checks collapsed", async ({ page }) => {
  await page.route("**/api/public/test-verify-creds/api/primary-key", async (route) => {
    await route.fulfill({ json: { data: { public_key_pem: "test-key", algorithm: "ECDSA" } } })
  })
  await page.goto("/verify-certificate")
  await page.locator('input[type="file"]').setInputFiles({ name: "invalid.pdf", mimeType: "application/pdf", buffer: samplePdf })

  await expect(page.getByRole("heading", { name: /Certificate is invalid|无效证书/ })).toBeVisible()
  await expect(page.locator(".verification-checks")).toContainText(/Failed|未通过/)
  const details = page.locator("details.verification-details")
  await expect(details).not.toHaveAttribute("open")
  await details.locator("summary").click()
  await expect(details).toHaveAttribute("open", "")
  await expect(details).toContainText(/ByteRange/)
})

test("reports an unavailable service without calling the certificate invalid", async ({ page }) => {
  await page.route("**/api/public/test-verify-creds/api/primary-key", async (route) => {
    await route.fulfill({ status: 503, json: { message: "Service unavailable" } })
  })
  await page.goto("/verify-certificate")
  await page.locator('input[type="file"]').setInputFiles({ name: "certificate.pdf", mimeType: "application/pdf", buffer: samplePdf })

  await expect(page.getByRole("heading", { name: /Verification unavailable|暂时无法完成查验/ })).toBeVisible()
  await expect(page.getByRole("heading", { name: /Certificate is invalid|无效证书/ })).toHaveCount(0)
})
