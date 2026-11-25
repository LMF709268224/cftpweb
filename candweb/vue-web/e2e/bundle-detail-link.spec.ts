import { expect, test } from "@playwright/test"
import { installCandidateApiMocks, seedAuthenticatedCandidate } from "./support/candidate"

test("product details open separately and empty links stay hidden", async ({ page, context }) => {
  await seedAuthenticatedCandidate(page)
  const detailsURL = "https://example.test/course-details"
  await context.route(detailsURL, route => route.fulfill({ contentType: "text/html", body: "<h1>Course details</h1>" }))
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/mall/bundles") {
      return { data: { bundles: [
        { bundle_id: "with-link", name: "Course with details", detail_url: detailsURL, description: "Course introduction", bundle_item_types: ["pipeline"], pipeline_id: "pipeline-1", is_pipeline_bundle: true, display_amount_min: 100, display_amount_max: 100, display_currency: "USD" },
        { bundle_id: "without-link", name: "Course without details", detail_url: "", description: "Course introduction", bundle_item_types: ["pipeline"], pipeline_id: "pipeline-2", is_pipeline_bundle: true, display_amount_min: 100, display_amount_max: 100, display_currency: "USD" },
      ] } }
    }
    return undefined
  })
  await page.goto("/certifications")
  const card = page.locator('[data-bundle-id="with-link"]')
  const link = card.locator(`a[href="${detailsURL}"]`)
  await expect(link).toBeVisible()
  await expect(link).toHaveAttribute("target", "_blank")
  await expect(link).toHaveAttribute("rel", "noopener noreferrer")
  await expect(page.locator('[data-bundle-id="without-link"]').getByRole("link")).toHaveCount(0)
  const popupPromise = page.waitForEvent("popup")
  await link.click()
  const popup = await popupPromise
  await expect(popup).toHaveURL(detailsURL)
  await expect(page).toHaveURL(/\/certifications$/)
})
