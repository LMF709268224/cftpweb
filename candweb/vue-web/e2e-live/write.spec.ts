import { expect, test, type APIRequestContext } from "@playwright/test"
import { liveEnvironment } from "./support/live"

type APIEnvelope<T> = {
  code?: number
  message?: string
  data?: T
}

type CandidateProfile = Record<string, unknown> & {
  bio?: string
}

function profileUpdatePayload(profile: CandidateProfile, bio: string) {
  return {
    email: String(profile.email || ""),
    display_name: String(profile.display_name || ""),
    first_name: String(profile.first_name || ""),
    last_name: String(profile.last_name || ""),
    phone_country_code: String(profile.phone_country_code || ""),
    phone: String(profile.phone || ""),
    home_phone: String(profile.home_phone || ""),
    country: String(profile.country || ""),
    province: String(profile.province || ""),
    city: String(profile.city || ""),
    address: String(profile.address_text || ""),
    postal_code: String(profile.postal_code || ""),
    affiliation: String(profile.affiliation || ""),
    title: String(profile.title || ""),
    real_name: String(profile.real_name || ""),
    bio,
    gender: String(profile.gender || ""),
    birthday: String(profile.birthday || ""),
    education: String(profile.education || ""),
  }
}

async function requestJSON<T>(
  request: APIRequestContext,
  method: "GET" | "POST" | "PUT",
  url: string,
  data?: unknown,
) {
  const response = await request.fetch(url, {
    method,
    data,
    headers: { Accept: "application/json" },
  })
  const payload = await response.json().catch(() => null) as APIEnvelope<T> | null
  expect(response.status(), `${method} ${new URL(url).pathname}`).toBeGreaterThanOrEqual(200)
  expect(response.status(), `${method} ${new URL(url).pathname}`).toBeLessThan(300)
  expect(payload?.code, `${method} ${new URL(url).pathname} response code`).toBe(200)
  return payload?.data as T
}

test.describe("candidate controlled live writes", () => {
  test.skip(
    process.env.E2E_LIVE_WRITE_ENABLED !== "true",
    "Set E2E_LIVE_WRITE_ENABLED=true only for an explicitly approved live write run.",
  )

  test("candidate profile can be updated and restored", async ({ page }) => {
    const environment = liveEnvironment()
    const request = page.context().request
    const meURL = new URL("/api/user/me", environment.baseURL).toString()
    const profileURL = new URL("/api/user/profile", environment.baseURL).toString()
    const original = await requestJSON<CandidateProfile>(request, "GET", meURL)
    const originalBio = String(original.bio || "")
    const marker = `e2e-write-check-${Date.now()}`

    try {
      await requestJSON(request, "PUT", profileURL, profileUpdatePayload(original, marker))
      await expect.poll(async () => {
        const current = await requestJSON<CandidateProfile>(request, "GET", meURL)
        return String(current.bio || "")
      }).toBe(marker)
    } finally {
      await requestJSON(request, "PUT", profileURL, profileUpdatePayload(original, originalBio))
    }

    await expect.poll(async () => {
      const restored = await requestJSON<CandidateProfile>(request, "GET", meURL)
      return String(restored.bio || "")
    }).toBe(originalBio)
  })

  test("dedicated bundle order can be created and cancelled", async ({ page }) => {
    const bundleID = process.env.E2E_LIVE_WRITE_BUNDLE_ID?.trim()
    test.skip(!bundleID, "E2E_LIVE_WRITE_BUNDLE_ID is not configured for a cancellable test bundle.")

    const environment = liveEnvironment()
    const request = page.context().request
    const createURL = new URL(`/api/mall/bundles/${encodeURIComponent(bundleID!)}/purchase`, environment.baseURL).toString()
    const cancelURL = new URL("/api/orders/cancel", environment.baseURL).toString()
    let bundleOrderID = ""

    try {
      const order = await requestJSON<{
        bundle_order_ulid?: string
        order_status?: string
        reused_existing?: boolean
      }>(request, "POST", createURL, {
        payment_mode: "FULL_PIPELINE",
        selected_exemptions_json: "[]",
      })
      bundleOrderID = String(order.bundle_order_ulid || "").trim()

      expect(bundleOrderID, "created bundle order ID").not.toBe("")
      expect(String(order.order_status || "").toUpperCase()).not.toMatch(/COMPLETED|PAID|SUCCESS/)
    } finally {
      if (bundleOrderID) {
        const cancelled = await requestJSON<{
          success?: boolean
          biz_ref_ulid?: string
          status?: string
        }>(request, "POST", cancelURL, {
          biz_type: "PIPELINE_BUNDLE",
          biz_ref_ulid: bundleOrderID,
        })
        expect(cancelled.success).toBe(true)
        expect(cancelled.biz_ref_ulid).toBe(bundleOrderID)
      }
    }
  })
})
