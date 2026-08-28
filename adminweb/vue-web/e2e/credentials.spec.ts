import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const credentialSummary = {
  cred_def_ulid: "credential-1",
  name: "Regression Credential",
  category: "Certification",
}

const credentialDetail = {
  ...credentialSummary,
  description: "Read-only regression definition",
  respath: "/gcc/credential/regression",
  acquisition_method: "Pass the regression assessment",
  file_constraints: [
    {
      name: "Evidence PDF",
      type: 2,
      is_required: true,
    },
  ],
  attachments: [
    {
      attachment_id: "attachment-1",
      name: "Regression Reference",
      description: "Read this reference before applying.",
      file_name: "reference.pdf",
      file_type: 2,
      file_ext: "pdf",
      file_size: 2048,
      file_hash: "reference-hash",
      file_key: "definitions/credential-1/reference.pdf",
      download_url: "https://downloads.example/reference.pdf",
    },
  ],
}

async function installCredentialReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/credentials/definitions") {
      return { data: { definitions: [credentialSummary] } }
    }
    if (method === "GET" && pathname === "/api/credentials/definitions/credential-1") {
      return { data: credentialDetail }
    }
    return undefined
  })
}

test("credential definition list renders returned read-only data", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installCredentialReadMocks(page, requests)

  await page.goto("/credentials")

  await expect(page.getByText("Regression Credential").first()).toBeVisible()
  await expect(page.getByText("认证资格").first()).toBeVisible()
  expect(requests).toContain("GET /api/credentials/definitions")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("credential definition detail displays metadata without a mutation", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installCredentialReadMocks(page, requests)
  await page.goto("/credentials")

  await page.getByRole("button", { name: "查看详情" }).click()
  const detailDialog = page.getByRole("dialog", { name: "Regression Credential" })
  await expect(detailDialog.getByText("/gcc/credential/regression", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText("Read-only regression definition", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText("Evidence PDF", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText("Regression Reference", { exact: true })).toBeVisible()
  await expect(detailDialog.getByRole("link", { name: "下载" })).toHaveAttribute("href", "https://downloads.example/reference.pdf")

  expect(requests).toContain("GET /api/credentials/definitions/credential-1")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("credential definition attachment is uploaded directly and saved as a complete list", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  let saved = false
  let uploadRequestBody: Record<string, unknown> | undefined
  let updateRequestBody: Record<string, unknown> | undefined

  page.on("request", (request) => {
    const pathname = new URL(request.url()).pathname
    if (request.method() === "POST" && pathname.endsWith("/attachments/upload-url")) uploadRequestBody = request.postDataJSON()
    if (request.method() === "PUT" && pathname.endsWith("/attachments")) updateRequestBody = request.postDataJSON()
  })
  await page.route("https://uploads.example/**", (route) => route.fulfill({ status: 200, body: "" }))
  await installAdminApiMocks(page, ({ method, pathname }) => {
    if (method === "GET" && pathname === "/api/credentials/definitions") return { data: { definitions: [credentialSummary] } }
    if (method === "GET" && pathname === "/api/credentials/definitions/credential-1") {
      return {
        data: saved
          ? {
              ...credentialDetail,
              attachments: [
                ...credentialDetail.attachments,
                {
                  name: "Work Experience Template",
                  description: "Complete and sign this template.",
                  file_name: "work-experience.docx",
                  file_type: 8,
                  file_ext: "docx",
                  file_size: 12,
                  file_hash: "mocked-in-browser",
                  file_key: "definitions/credential-1/work-experience.docx",
                },
              ],
            }
          : credentialDetail,
      }
    }
    if (method === "POST" && pathname === "/api/credentials/definitions/credential-1/attachments/upload-url") {
      return {
        data: {
          upload_url: "https://uploads.example/work-experience.docx",
          file_key: "definitions/credential-1/work-experience.docx",
          signed_headers: { "content-type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document" },
        },
      }
    }
    if (method === "PUT" && pathname === "/api/credentials/definitions/credential-1/attachments") {
      saved = true
      return { data: { success: true } }
    }
    return undefined
  })

  await page.goto("/credentials")
  await page.getByRole("button", { name: "查看详情" }).click()
  const dialog = page.getByRole("dialog", { name: "Regression Credential" })
  await dialog.getByLabel("附件显示名称").fill("Work Experience Template")
  await dialog.getByLabel("文件类型").selectOption("8")
  await dialog.getByLabel("附件说明 / 填写指引").fill("Complete and sign this template.")
  await dialog.getByLabel("选择文件", { exact: true }).setInputFiles({
    name: "work-experience.docx",
    mimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    buffer: Buffer.from("docx-content"),
  })
  await dialog.getByRole("button", { name: "上传并保存" }).click()

  await expect(dialog.getByText("Work Experience Template", { exact: true })).toBeVisible()
  expect(uploadRequestBody).toMatchObject({
    file_name: "work-experience.docx",
    file_ext: "docx",
    content_type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  })
  expect(updateRequestBody).toMatchObject({
    attachments: [
      expect.objectContaining({ file_key: "definitions/credential-1/reference.pdf" }),
      expect.objectContaining({
        name: "Work Experience Template",
        file_name: "work-experience.docx",
        file_type: 8,
        file_key: "definitions/credential-1/work-experience.docx",
      }),
    ],
  })
})
