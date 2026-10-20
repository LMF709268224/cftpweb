import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const schema = JSON.stringify({
  type: "object",
  required: ["Name", "Email", "Count", "SendCopy"],
  properties: {
    Name: { type: "string", minLength: 2 },
    Email: { type: "string", format: "email" },
    Count: { type: "integer", minimum: 1 },
    SendCopy: { type: "boolean" },
  },
})
const messageTemplate = { path: "/msg/regression", title_tpl: "Hello {{Name}}", content_tpl: "### **{{Name}}**\n\n{{Email}} {{Count}} {{SendCopy}}", version: 3, parameter_schema: schema }
const mailTemplate = { path: "/mail/regression", name: "Regression mail", subject_template: "Hello {{Name}}", html_body: "<p>{{Name}} {{Email}} {{Count}} {{SendCopy}}</p>", parameter_schema: schema }

async function setup(page: Page, lang: "zh" | "en" = "zh") {
  await seedAuthenticatedAdmin(page, lang)
  await installAdminApiMocks(page, ({ method, pathname }) => {
    if (method !== "GET") return undefined
    if (pathname === "/api/user/list") return { data: { users: [{ id: "candidate-1", name: "Regression Candidate", email: "candidate@example.test" }] } }
    if (pathname === "/api/messages/templates") return { data: { templates: [messageTemplate], total: 1 } }
    if (pathname === "/api/messages/templates/detail") return { data: messageTemplate }
    if (pathname === "/api/mails/templates") return { data: { templates: [mailTemplate], total: 1 } }
    if (pathname === "/api/mails/templates/detail") return { data: mailTemplate }
    return undefined
  })
}

async function fillParameters(page: Page) {
  await page.getByRole("textbox", { name: "Name", exact: true }).fill("Test User")
  await page.getByRole("textbox", { name: "Email", exact: true }).fill("test@example.test")
  await page.getByRole("spinbutton", { name: "Count", exact: true }).fill("2")
  await page.getByRole("combobox", { name: "SendCopy", exact: true }).selectOption("false")
}

test("message parameters validate required fields, formats and types before keeping the send contract", async ({ page }) => {
  await setup(page)
  const bodies: Record<string, unknown>[] = []
  await page.route("**/api/messages/send", async route => {
    bodies.push(route.request().postDataJSON())
    await route.fulfill({ json: { code: 200, data: { count: 1 } } })
  })
  await page.goto("/messages")
  await page.getByRole("combobox", { name: "模板", exact: true }).selectOption(messageTemplate.path)
  await page.getByRole("checkbox", { name: /Regression Candidate/ }).check()
  const send = page.getByRole("button", { name: "发送站内信", exact: true }).last()
  await expect(send).toBeEnabled()
  await send.click()
  await expect(page.getByRole("alert")).toContainText("Name")
  expect(bodies).toHaveLength(0)
  await fillParameters(page)
  await page.getByRole("textbox", { name: "Email", exact: true }).fill("invalid")
  await send.click()
  await expect(page.getByRole("alert")).toContainText("Email")
  expect(bodies).toHaveLength(0)
  await page.getByRole("textbox", { name: "Email", exact: true }).fill("test@example.test")
  await page.getByRole("spinbutton", { name: "Count" }).fill("1.5")
  await send.click()
  await expect(page.getByRole("alert")).toContainText("Count")
  expect(bodies).toHaveLength(0)
  await page.getByRole("spinbutton", { name: "Count" }).fill("2")
  await send.click()
  await expect.poll(() => bodies.length).toBe(1)
  expect(bodies[0]).toEqual({ user_ids: ["candidate-1"], template_path: messageTemplate.path, msg_type: 1, payload: expect.any(String) })
  expect(JSON.parse(String(bodies[0]!.payload))).toEqual({ Name: "Test User", Email: "test@example.test", Count: 2, SendCopy: false })
})

test("message templates retain the original text editor and update fields without variable insertion controls", async ({ page }) => {
  await setup(page)
  const updates: Record<string, unknown>[] = []
  await page.route("**/api/messages/templates", async route => {
    if (route.request().method() === "GET") return route.fallback()
    updates.push(route.request().postDataJSON())
    await route.fulfill({ json: { code: 200, data: {} } })
  })
  await page.goto("/messages")
  await page.getByRole("button", { name: "模板管理", exact: true }).click()
  await page.getByRole("button", { name: "编辑", exact: true }).click()
  const dialog = page.getByRole("dialog", { name: "编辑模板" })
  const content = dialog.getByRole("textbox", { name: /内容模板/ })
  await expect(dialog.getByRole("combobox", { name: "模板变量" })).toHaveCount(0)
  await expect(dialog.getByRole("button", { name: "插入变量", exact: true })).toHaveCount(0)
  await content.fill("{{Name}} ### Keep **Markdown**")
  await dialog.getByRole("button", { name: "保存模板", exact: true }).click()
  await expect.poll(() => updates.length).toBe(1)
  expect(updates[0]).toMatchObject({ path: messageTemplate.path, title_tpl: messageTemplate.title_tpl, content_tpl: "{{Name}} ### Keep **Markdown**", parameter_schema: schema, current_version: 3 })
})

for (const lang of ["zh", "en"] as const) {
  for (const width of [390, 1440]) {
    test(`mail template preview validates and retries without sending (${lang}, ${width})`, async ({ page }) => {
      await page.setViewportSize({ width, height: 1000 })
      await setup(page, lang)
      const renders: Record<string, unknown>[] = []
      let sends = 0
      await page.route("**/api/mails/send", async route => { sends++; await route.fulfill({ json: { code: 200, data: {} } }) })
      await page.route("**/api/mails/templates/render", async route => {
        renders.push(route.request().postDataJSON())
        if (renders.length === 1) return route.fulfill({ status: 500, json: { message: "Preview unavailable" } })
        await route.fulfill({ json: { code: 200, data: { subject: "Hello Test User", html_body: "<h1>Hello Test User</h1>" } } })
      })
      await page.goto("/mails")
      await page.getByRole("combobox", { name: lang === "zh" ? "模板" : "Template", exact: true }).selectOption(mailTemplate.path)
      const preview = page.getByRole("button", { name: lang === "zh" ? "预览" : "Preview", exact: true })
      await expect(preview).toBeEnabled()
      await preview.click()
      await expect(page.getByRole("alert")).toContainText("Name")
      expect(renders).toHaveLength(0)
      await fillParameters(page)
      await preview.click()
      const dialog = page.getByRole("dialog", { name: lang === "zh" ? "邮件预览" : "Email Preview" })
      await expect(dialog.getByRole("alert")).toContainText("Preview unavailable")
      await dialog.getByRole("button", { name: lang === "zh" ? "重试" : "Retry" }).click()
      await expect(dialog.locator("iframe")).toHaveAttribute("srcdoc", "<h1>Hello Test User</h1>")
      await expect(dialog.locator("iframe")).toHaveAttribute("sandbox", "")
      await expect(dialog).toContainText("Hello Test User")
      expect(renders).toHaveLength(2)
      expect(renders[1]).toEqual({ template_path: mailTemplate.path, template_params_json: expect.any(String), subject_params_json: expect.any(String) })
      expect(renders[1]!.template_params_json).toBe(renders[1]!.subject_params_json)
      expect(JSON.parse(String(renders[1]!.template_params_json))).toEqual({ Name: "Test User", Email: "test@example.test", Count: 2, SendCopy: false })
      expect(sends).toBe(0)
      await page.keyboard.press("Escape")
      await expect(dialog).toBeHidden()
    })
  }
}

test("closing a pending preview prevents late results from reopening it", async ({ page }) => {
  await setup(page)
  let release!: () => void
  const gate = new Promise<void>(resolve => { release = resolve })
  let requested = false
  await page.route("**/api/mails/templates/render", async route => {
    requested = true
    await gate
    await route.fulfill({ json: { code: 200, data: { subject: "Late", html_body: "<p>Late</p>" } } })
  })
  await page.goto("/mails")
  await page.getByRole("combobox", { name: "模板", exact: true }).selectOption(mailTemplate.path)
  await fillParameters(page)
  await page.getByRole("button", { name: "预览", exact: true }).click()
  await expect.poll(() => requested).toBe(true)
  const dialog = page.getByRole("dialog", { name: "邮件预览" })
  await expect(dialog.getByRole("status")).toBeVisible()
  await page.keyboard.press("Escape")
  const response = page.waitForResponse("**/api/mails/templates/render")
  release()
  await response
  await expect(dialog).toBeHidden()
  await expect(page.getByRole("button", { name: "预览", exact: true })).toBeEnabled()
})

test("complex parameters remain editable as JSON without losing nested values", async ({ page }) => {
  await setup(page)
  await page.route("**/api/messages/templates/detail?**", route => route.fulfill({ json: { code: 200, data: { ...messageTemplate, title_tpl: "Complex", content_tpl: "{{if .User}}{{.User.Name}}{{end}}", parameter_schema: JSON.stringify({ type: "object", properties: { User: { type: "object", properties: { Name: { type: "string" } } } } }) } } }))
  let sent: Record<string, unknown> | undefined
  await page.route("**/api/messages/send", async route => { sent = route.request().postDataJSON(); await route.fulfill({ json: { code: 200, data: { count: 1 } } }) })
  await page.goto("/messages")
  await page.getByRole("combobox", { name: "模板", exact: true }).selectOption(messageTemplate.path)
  await expect(page.getByText(/部分参数规则/)).toBeVisible()
  await expect(page.getByRole("textbox", { name: "end", exact: true })).toHaveCount(0)
  await page.getByRole("textbox", { name: "Payload JSON", exact: true }).fill('{"User":{"Name":"Original"},"Count":0}')
  await page.getByRole("checkbox", { name: /Regression Candidate/ }).check()
  await page.getByRole("button", { name: "发送站内信", exact: true }).last().click()
  await expect.poll(() => sent).toBeTruthy()
  expect(JSON.parse(String(sent!.payload))).toEqual({ User: { Name: "Original" }, Count: 0 })
})

for (const channel of ["messages", "mails"] as const) {
  test(`${channel}: switching templates ignores stale details and displays load failures for retry`, async ({ page }) => {
    await setup(page)
    const template = channel === "messages" ? messageTemplate : mailTemplate
    await page.route(new RegExp(`/api/${channel}/templates(?:\\?.*)?$`), route => route.fulfill({ json: { code: 200, data: { templates: [template, { ...template, path: "/second" }], total: 2 } } }))
    let release!: () => void
    const gate = new Promise<void>(resolve => { release = resolve })
    let requested = false
    let secondLoads = 0
    await page.route(`**/api/${channel}/templates/detail?**`, async route => {
      const path = new URL(route.request().url()).searchParams.get("path")
      if (path === template.path) { requested = true; await gate }
      else if (++secondLoads === 1) return route.fulfill({ status: 500, json: { message: "Template unavailable" } })
      await route.fulfill({ json: { code: 200, data: { ...template, path, parameter_schema: JSON.stringify({ type: "object", properties: path === template.path ? { OldOnly: { type: "string" } } : { NewOnly: { type: "string" } } }) } } })
    })
    await page.goto(`/${channel}`)
    const select = page.getByRole("combobox", { name: "模板", exact: true })
    await select.selectOption(template.path)
    await expect.poll(() => requested).toBe(true)
    await select.selectOption("/second")
    await expect(page.getByRole("alert")).toContainText("Template unavailable")
    await expect(page.getByRole("button", { name: channel === "messages" ? "发送站内信" : "发送邮件", exact: true }).last()).toBeDisabled()
    await page.getByRole("alert").getByRole("button", { name: "刷新模板" }).click()
    await expect(page.getByRole("textbox", { name: "NewOnly", exact: true })).toBeVisible()
    const response = page.waitForResponse(res => res.url().includes("templates/detail") && res.url().includes(encodeURIComponent(template.path)))
    release()
    await response
    await expect(page.getByRole("textbox", { name: "NewOnly", exact: true })).toBeVisible()
    await expect(page.getByRole("textbox", { name: "OldOnly", exact: true })).toHaveCount(0)
  })
}

test("mail parameter forms preserve the existing send request fields", async ({ page }) => {
  await setup(page)
  let sent: Record<string, unknown> | undefined
  await page.route("**/api/mails/send", async route => { sent = route.request().postDataJSON(); await route.fulfill({ json: { code: 200, data: {} } }) })
  await page.goto("/mails")
  await page.getByRole("combobox", { name: "模板", exact: true }).selectOption(mailTemplate.path)
  await fillParameters(page)
  await page.getByRole("textbox", { name: "邮件主题", exact: true }).fill("Original subject override")
  await page.getByRole("checkbox", { name: /Regression Candidate/ }).check()
  await page.getByRole("button", { name: "发送邮件", exact: true }).last().click()
  await expect.poll(() => sent).toBeTruthy()
  expect(sent).toEqual({ to_email: "candidate@example.test", to_name: "Regression Candidate", subject: "Original subject override", template_path: mailTemplate.path, payload: expect.any(String), html_body: "", plain_body: "" })
  expect(JSON.parse(String(sent!.payload))).toEqual({ Name: "Test User", Email: "test@example.test", Count: 2, SendCopy: false })
})

test("templates without type rules preserve values from the existing JSON editor", async ({ page }) => {
  await setup(page)
  await page.route("**/api/messages/templates/detail?**", route => route.fulfill({ json: { code: 200, data: { ...messageTemplate, parameter_schema: "{}" } } }))
  let sent: Record<string, unknown> | undefined
  await page.route("**/api/messages/send", async route => { sent = route.request().postDataJSON(); await route.fulfill({ json: { code: 200, data: { count: 1 } } }) })
  await page.goto("/messages")
  await page.getByRole("combobox", { name: "模板", exact: true }).selectOption(messageTemplate.path)
  await expect(page.getByRole("textbox", { name: "Name", exact: true })).toBeVisible()
  await page.getByText("JSON 编辑", { exact: true }).click()
  const original = { Name: 7, Email: null, Count: 0, SendCopy: false, Extra: { keep: true } }
  await page.getByRole("textbox", { name: "Payload JSON", exact: true }).fill(JSON.stringify(original))
  await page.getByRole("checkbox", { name: /Regression Candidate/ }).check()
  await page.getByRole("button", { name: "发送站内信", exact: true }).last().click()
  await expect.poll(() => sent).toBeTruthy()
  expect(JSON.parse(String(sent!.payload))).toEqual(original)
})
