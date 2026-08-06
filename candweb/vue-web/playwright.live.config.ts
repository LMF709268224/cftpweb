import { defineConfig, devices } from "@playwright/test"

const baseURL = process.env.E2E_CANDIDATE_BASE_URL?.trim() || "http://127.0.0.1.invalid"

export default defineConfig({
  testDir: "./e2e-live",
  outputDir: "test-results/live",
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  timeout: 60_000,
  expect: {
    timeout: 20_000,
  },
  reporter: [
    ["list"],
    ["html", { open: "never", outputFolder: "playwright-report-live" }],
  ],
  use: {
    baseURL,
    ignoreHTTPSErrors: false,
    trace: "off",
    screenshot: "off",
    video: "off",
  },
  projects: [
    {
      name: "live-auth",
      testMatch: /auth\.setup\.ts/,
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "live-readonly",
      testMatch: /read-only\.spec\.ts/,
      dependencies: ["live-auth"],
      use: {
        ...devices["Desktop Chrome"],
        storageState: "test-results/live-auth/candidate.json",
      },
    },
    {
      name: "live-write",
      testMatch: /write\.spec\.ts/,
      dependencies: ["live-auth"],
      use: {
        ...devices["Desktop Chrome"],
        storageState: "test-results/live-auth/candidate.json",
      },
    },
    {
      name: "live-journey",
      testMatch: /journey\.spec\.ts/,
      dependencies: ["live-auth"],
      use: {
        ...devices["Desktop Chrome"],
        storageState: "test-results/live-auth/candidate.json",
      },
    },
  ],
})
