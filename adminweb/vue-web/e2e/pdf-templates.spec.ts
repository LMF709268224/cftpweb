import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const templateSummary = {
  template_ulid: "template-1",
  name: "Regression Certificate",
  description: "Read-only regression template",
  version: 2,
  created_at: "2026-08-11T00:00:00Z",
}

async function installPdfTemplateReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/pdf-templates") {
      return { data: { templates: [templateSummary] } }
    }
    if (method === "GET" && pathname === "/api/pdf-templates/detail") {
      return {
        data: {
          ...templateSummary,
          html_template: "<main>Regression Certificate</main>",
          parameter_schema: '{"type":"object"}',
        },
      }
    }
    if (method === "GET" && pathname === "/api/pdf-templates/template-1/translations") {
      return { data: { translations: {} } }
    }
    return undefined
  })
}

test("PDF template list renders the returned read-only summary", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installPdfTemplateReadMocks(page, requests)

  await page.goto("/pdf-templates")

  await expect(page.getByText("Regression Certificate", { exact: true })).toBeVisible()
  await expect(page.getByText("Read-only regression template", { exact: true })).toBeVisible()
  await expect(page.getByText("v2", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/pdf-templates")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("PDF template detail displays HTML and schema without editing", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installPdfTemplateReadMocks(page, requests)
  await page.goto("/pdf-templates")

  await page.getByRole("button", { name: "查看详情" }).click()
  const dialog = page.getByRole("dialog", { name: "模板详情" })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText("<main>Regression Certificate</main>", { exact: true })).toBeVisible()
  await expect(dialog.getByText('{"type":"object"}', { exact: true })).toBeVisible()
  const preview = dialog.locator('iframe[title="HTML 预览"]')
  await expect(preview).toHaveAttribute("srcdoc", "<main>Regression Certificate</main>")
  await expect(preview).toHaveCSS("width", "1123px")
  await expect(preview).toHaveCSS("height", "794px")
  const previewLayout = await preview.evaluate((frame) => {
    const viewport = frame.parentElement
    if (!viewport) throw new Error("Missing preview viewport")
    const frameRect = frame.getBoundingClientRect()
    return {
      frameWidth: frameRect.width,
      scale: new DOMMatrix(getComputedStyle(frame).transform).a,
      viewportWidth: viewport.clientWidth,
    }
  })
  expect(previewLayout.scale).toBeLessThan(1)
  expect(Math.abs(previewLayout.frameWidth - previewLayout.viewportWidth)).toBeLessThan(1)

  expect(requests).toContain("GET /api/pdf-templates/detail")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
