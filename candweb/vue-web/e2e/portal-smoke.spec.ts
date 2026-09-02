import { expect, test } from "@playwright/test";
import {
  installCandidateApiMocks,
  seedAuthenticatedCandidate,
  type ApiMockContext,
} from "./support/candidate";

const portalPages = [
  { path: "/dashboard", heading: "欢迎来到门户" },
  { path: "/certifications", heading: "商城" },
  { path: "/my-certifications", heading: "我的认证" },
  { path: "/exams", heading: "考试" },
  { path: "/records", heading: "档案" },
  { path: "/resource-packs", heading: "资源包" },
  { path: "/credentials", heading: "资格申请" },
  { path: "/certificates", heading: "证书" },
  { path: "/membership", heading: "会员" },
  { path: "/orders", heading: "订单" },
  { path: "/messages", heading: "消息" },
  { path: "/settings", heading: "设置" },
];

function emptyPortalResponse({ pathname }: ApiMockContext) {
  if (pathname === "/api/pipeline") return { data: { list: [] } };
  if (pathname === "/api/certificates") return { data: { certificates: [] } };
  if (pathname === "/api/exams") {
    return {
      data: {
        exams: [],
        total: 0,
        total_label: "0",
        total_pages: 0,
        has_more: false,
        next_cursor: "",
        prev_cursor: "",
      },
    };
  }
  if (pathname === "/api/resource-packs") {
    return { data: { packs: [], next_page_token: "" } };
  }
  if (pathname === "/api/orders") {
    return {
      data: {
        orders: [],
        total_orders: 0,
        total_label: "0",
        total_pages: 0,
        has_more: false,
        next_cursor: "",
        prev_cursor: "",
      },
    };
  }
  if (pathname === "/api/mall/bundles") return { data: { bundles: [] } };
  if (pathname === "/api/credentials/definitions")
    return { data: { definitions: [] } };
  if (pathname === "/api/credentials/applications") {
    return {
      data: {
        applications: [],
        total: 0,
        total_label: "0",
        total_pages: 0,
        has_more: false,
        next_cursor: "",
        prev_cursor: "",
      },
    };
  }
  if (pathname === "/api/messages") {
    return {
      data: {
        messages: [],
        has_more: false,
        next_cursor: "",
        prev_cursor: "",
      },
    };
  }
  if (pathname === "/api/membership/plans") return { data: { plans: [] } };
  if (pathname === "/api/membership/history") {
    return {
      data: {
        memberships: [],
        total: 0,
        total_pages: 0,
        has_more: false,
        next_cursor: "",
        prev_cursor: "",
      },
    };
  }
  if (pathname === "/api/membership/billings") {
    return {
      data: {
        billings: [],
        total: 0,
        total_pages: 0,
        has_more: false,
        next_cursor: "",
        prev_cursor: "",
      },
    };
  }
  return undefined;
}

for (const portalPage of portalPages) {
  test(`${portalPage.heading}页面可以在空数据状态下正常打开`, async ({
    page,
  }) => {
    const pageErrors: string[] = [];
    page.on("pageerror", (error) => pageErrors.push(error.message));

    await seedAuthenticatedCandidate(page);
    await installCandidateApiMocks(page, emptyPortalResponse);
    await page.goto(portalPage.path, { waitUntil: "domcontentloaded" });

    await expect(page).toHaveURL(
      new RegExp(`${portalPage.path.replaceAll("/", "\\/")}$`),
    );
    await expect(
      page
        .getByRole("heading", { name: portalPage.heading, exact: true })
        .first(),
    ).toBeVisible();
    await expect.poll(() => pageErrors).toEqual([]);
  });
}

test("公开商城按商品类型区分报名与成为会员文案", async ({ page }) => {
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname !== "/api/mall/bundles") return undefined;
    return {
      data: {
        bundles: [
          {
            bundle_id: "public-certification",
            pipeline_id: "public-pipeline",
            is_pipeline_bundle: true,
            name: "公开认证项目",
          },
          {
            bundle_id: "public-membership",
            membership_id: "public-membership",
            is_membership_bundle: true,
            name: "公开会员服务",
          },
        ],
      },
    };
  });

  await page.goto("/", { waitUntil: "domcontentloaded" });

  const certificationCard = page.locator("article.product-card").filter({ hasText: "公开认证项目" });
  const membershipCard = page.locator("article.product-card").filter({ hasText: "公开会员服务" });
  await expect(certificationCard.getByRole("button")).toContainText("去报名");
  await expect(membershipCard.getByRole("button")).toContainText("成为会员");

  await page.evaluate(() => {
    localStorage.setItem("app_lang", "en");
    window.dispatchEvent(new Event("lang_change"));
  });
  await expect(certificationCard.getByRole("button")).toContainText("Enroll Now");
  await expect(membershipCard.getByRole("button")).toContainText("Become a Member");
});

test("登录后商城区分认证报名与会员加入文案", async ({ page }) => {
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname !== "/api/mall/bundles") return undefined;
    return {
      data: {
        bundles: [
          {
            bundle_id: "candidate-certification",
            pipeline_id: "candidate-pipeline",
            is_pipeline_bundle: true,
            name: "登录后认证项目",
            eligibility: { can_purchase: true, blockers: [] },
          },
          {
            bundle_id: "candidate-membership",
            membership_id: "candidate-membership",
            is_membership_bundle: true,
            name: "登录后会员服务",
            eligibility: { can_purchase: true, blockers: [] },
          },
        ],
      },
    };
  });

  await page.goto("/certifications", { waitUntil: "domcontentloaded" });

  const certificationCard = page.locator('[data-bundle-id="candidate-certification"]');
  const membershipCard = page.locator('[data-bundle-id="candidate-membership"]');
  await expect(certificationCard).toContainText("去报名");
  await expect(membershipCard).toContainText("成为会员");

  await page.evaluate(() => {
    localStorage.setItem("app_lang", "en");
    window.dispatchEvent(new Event("lang_change"));
  });
  await expect(certificationCard).toContainText("Enroll Now");
  await expect(membershipCard).toContainText("Become a Member");
});

test("首页统计卡片可以跳转到对应页面", async ({ page }) => {
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, emptyPortalResponse);
  await page.goto("/dashboard", { waitUntil: "domcontentloaded" });

  const dashboardCards = [
    { key: "certificates", path: "/certificates" },
    { key: "exams", path: "/exams" },
    { key: "resourcePacks", path: "/resource-packs" },
  ];

  for (const card of dashboardCards) {
    await page.goto("/dashboard", { waitUntil: "domcontentloaded" });
    const link = page.getByTestId(`dashboard-card-${card.key}`);
    await expect(link).toHaveAttribute("href", card.path);
    await link.click();
    await expect(page).toHaveURL(new RegExp(`${card.path.replaceAll("/", "\\/")}$`));
  }
});

test("支付成功页在移动端不会因长 ID 横向溢出", async ({ page }) => {
  const longOrderId = "ORDER-MOBILE-REGRESSION-0123456789-ABCDEFGHIJKLMNOPQRSTUVWXYZ-0123456789";

  await page.setViewportSize({ width: 382, height: 739 });
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, emptyPortalResponse);
  await page.goto(`/checkout/success/${longOrderId}`, { waitUntil: "domcontentloaded" });

  const card = page.locator(".checkout-success-card");
  const orderId = page.locator(".checkout-success-id").first();

  await expect(card).toBeVisible();
  await expect(page.getByRole("link", { name: "查看我的认证" })).toBeVisible();
  await expect(page.getByRole("link", { name: "返回商城" })).toBeVisible();
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await expect.poll(() => orderId.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(true);

  const cardBox = await card.boundingBox();
  expect(cardBox).not.toBeNull();
  expect(cardBox!.width).toBeLessThanOrEqual(350);
  await expect.poll(() => card.evaluate((element) => getComputedStyle(element).paddingLeft)).toBe("20px");
});

test("资格申请详情提供官方模板预览与下载", async ({ page }) => {
  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/credentials/definitions") {
      return {
        data: {
          definitions: [
            {
              cred_def_ulid: "credential-with-template",
              name: "Work Experience Qualification",
              category: "Qualification",
              description: "Upload signed work experience evidence.",
              respath: "/gcreds/core/work-experience",
              file_constraints: [],
              attachments: [{
                attachment_id: "attachment-template-1",
                name: "工作经验证明模板",
                description: "下载填写并签字后再上传。",
                file_name: "work-experience-template.docx",
                file_ext: "docx",
                file_size: 4096,
                download_url: "https://downloads.example/work-experience-template.docx",
              }],
            },
            {
              cred_def_ulid: "credential-system-only",
              name: "System Managed Qualification",
              category: "Certification",
              respath: "/gcreds/credentials/system/internal-certification",
              file_constraints: [],
            },
          ],
        },
      }
    }
    if (pathname === "/api/credentials/applications") {
      return {
        data: {
          applications: [{
            app_ulid: "template-application",
            cred_def_ulid: "credential-with-template",
            status: "PENDING_UPLOAD",
          }],
          total: 1,
        },
      }
    }
    return undefined
  })

  await page.goto("/credentials?qual_ulids=credential-with-template", { waitUntil: "domcontentloaded" })
  const dialog = page.locator(".credentials-apply-dialog")
  await expect(dialog.getByText("模板与参考文件", { exact: true })).toBeVisible()
  await expect(dialog.getByText("工作经验证明模板", { exact: true })).toBeVisible()
  await expect(page.getByText("System Managed Qualification", { exact: true })).toHaveCount(0)
  await expect(dialog.getByRole("link", { name: "预览模板" })).toHaveAttribute(
    "href",
    "https://downloads.example/work-experience-template.docx",
  )
  await expect(dialog.getByRole("link", { name: "下载模板" })).toHaveAttribute(
    "href",
    "https://downloads.example/work-experience-template.docx",
  )
  await expect(dialog.getByRole("link", { name: "下载模板" })).toHaveAttribute("download", "work-experience-template.docx")
})

test("资格申请弹窗在移动端保持操作区可见", async ({ page }) => {
  await page.setViewportSize({ width: 382, height: 739 });
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/credentials/definitions") {
      return {
        data: {
          definitions: [{
            cred_def_ulid: "credential-mobile",
            name: "移动端资格申请",
            category: "Qualification",
            description: "用于验证材料较多时弹窗仍可正常操作。",
            file_constraints: Array.from({ length: 8 }, (_, index) => ({
              name: `证明材料 ${index + 1}`,
              type: "document",
              is_required: true,
            })),
          }],
        },
      };
    }
    if (pathname === "/api/credentials/applications") {
      return {
        data: {
          applications: [{
            app_ulid: "mobile-application",
            cred_def_ulid: "credential-mobile",
            status: "PENDING_UPLOAD",
          }],
          total: 1,
        },
      };
    }
    return undefined;
  });

  await page.goto("/credentials?qual_ulids=credential-mobile", { waitUntil: "domcontentloaded" });

  const dialog = page.locator(".credentials-apply-dialog");
  const scrollBody = page.locator(".credentials-apply-body");
  const actions = page.locator(".credentials-apply-actions");
  const closeButton = page.locator(".credentials-apply-close");

  await expect(dialog).toBeVisible();
  await expect(actions).toBeVisible();
  await expect.poll(() => scrollBody.evaluate((element) => element.scrollHeight > element.clientHeight)).toBe(true);

  const dialogBox = await dialog.boundingBox();
  const closeButtonBox = await closeButton.boundingBox();
  expect(dialogBox).not.toBeNull();
  expect(closeButtonBox).not.toBeNull();
  expect(dialogBox!.y).toBeGreaterThanOrEqual(0);
  expect(dialogBox!.y + dialogBox!.height).toBeLessThanOrEqual(739);
  expect(closeButtonBox!.width).toBeGreaterThanOrEqual(44);
  expect(closeButtonBox!.height).toBeGreaterThanOrEqual(44);
});
