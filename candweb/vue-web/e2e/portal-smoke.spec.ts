import { expect, test } from "@playwright/test";
import {
  candidateUser,
  installCandidateApiMocks,
  seedAuthenticatedCandidate,
  type ApiMockContext,
} from "./support/candidate";

const portalPages = [
  { path: "/dashboard", heading: "欢迎回来，Regression" },
  { path: "/certifications", heading: "认证与课程" },
  { path: "/my-certifications", heading: "我的专业认证考试" },
  { path: "/exams", heading: "考试" },
  { path: "/records", heading: "档案" },
  { path: "/resource-packs", heading: "专属合集" },
  { path: "/credentials", heading: "资格认证申请" },
  { path: "/certificates", heading: "认证证书" },
  { path: "/membership", heading: "会员资格" },
  { path: "/orders", heading: "购买记录" },
  { path: "/messages", heading: "通知" },
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

test("首页展示学习进度、最新 Insights 和现有认证课程", async ({ page }) => {
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/pipeline") return { data: { list: [
      { pipeline_ulid: "owned-1", pipeline_cc_ulid: "config-1", pipeline_name: "CFtP", status: "PIPELINE_STATUS_RUNNING", current_stage_name: "Level 1", progress_available: true, progress: 68, started_at: "2026-10-18T00:00:00Z" },
      { pipeline_ulid: "owned-done", pipeline_cc_ulid: "config-done", pipeline_name: "Completed", status: "PIPELINE_STATUS_COMPLETED", progress_available: true, progress: 100 },
    ] } };
    if (pathname === "/api/resource-packs") return { data: { packs: [
      { pack_id: "insights-pack", title: "Insights", respath: "/insights" },
    ], next_page_token: "" } };
    if (pathname === "/api/resource-packs/insights-pack/files") return { data: { files: [
      { file_id: "older", title: "Older report", file_type: 2, created_at: "2026-10-01T00:00:00Z" },
      { file_id: "newer", title: "New report", file_type: 2, created_at: "2026-10-21T00:00:00Z" },
      { file_id: "middle", title: "Middle report", file_type: 2, created_at: "2026-10-12T00:00:00Z" },
    ], next_page_token: "" } };
    if (pathname === "/api/mall/bundles") return { data: { bundles: [
      { bundle_id: "cert-1", pipeline_id: "config-1", is_pipeline_bundle: true, name: "CFtP" },
      { bundle_id: "member-1", is_membership_bundle: true, name: "Membership" },
      { bundle_id: "cert-2", pipeline_id: "config-2", is_pipeline_bundle: true, name: "CFtA" },
      { bundle_id: "course-1", pipeline_id: "config-3", is_pipeline_bundle: true, name: "Course" },
    ] } };
    return emptyPortalResponse({ pathname } as ApiMockContext);
  });

  await page.goto("/dashboard", { waitUntil: "domcontentloaded" });

  await expect(page.getByTestId("dashboard-learning-card")).toContainText("68%");
  await expect(page.getByTestId("dashboard-learning-card")).toHaveCount(1);
  await expect(page.getByTestId("dashboard-learning-card")).toHaveAttribute("href", "/certifications/config-1");
  await expect(page.getByTestId("dashboard-insight-card")).toHaveCount(2);
  await expect(page.getByTestId("dashboard-insight-card").first()).toContainText("New report");
  await expect(page.getByTestId("dashboard-insight-card").first()).toHaveAttribute("href", "/resource-pack-files/newer/preview");
  await expect(page.getByTestId("dashboard-product-card")).toHaveCount(3);
  await expect(page.getByTestId("dashboard-product-card").last()).toHaveAttribute("href", "/certifications/config-3");
  await expect(page.getByText("Membership", { exact: true })).toHaveCount(0);
  await expect(page.getByTestId("dashboard-account-status")).toHaveCount(0);

  await page.setViewportSize({ width: 390, height: 844 });
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
});

test("首页没有学习记录时显示开始学习入口", async ({ page }) => {
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, emptyPortalResponse);

  await page.goto("/dashboard", { waitUntil: "domcontentloaded" });

  await expect(page.getByText("还没有开始学习")).toBeVisible();
  await expect(page.getByRole("link", { name: "开始学习", exact: true })).toHaveAttribute("href", "/certifications");
});

test("首页不展示微服务未能确认的用户状态", async ({ page }) => {
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/user/me") {
      return {
        data: {
          name: "Regression Candidate",
          account_status: {
            membership: { available: false, is_member: false },
            certification: { available: false, is_candidate: false, purchase_count: 0, programs: [] },
            qualification: { available: false, has_qualification: false, credential_count: 0 },
          },
        },
      };
    }
    return emptyPortalResponse({ pathname } as ApiMockContext);
  });

  await page.goto("/dashboard", { waitUntil: "domcontentloaded" });

  await expect(page.getByTestId("dashboard-account-status")).toHaveCount(0);
});

for (const scenario of [
  { name: "已购课程优先学习", purchaseCount: 1, isMember: false, label: "开始学习", href: "/my-certifications" },
  { name: "无课程时引导入会", purchaseCount: 0, isMember: false, label: "成为会员", href: "/membership" },
  { name: "已有会员时浏览商品", purchaseCount: 0, isMember: true, label: "探索课程与认证", href: "/certifications" },
]) {
  test(`首页下一步：${scenario.name}`, async ({ page }) => {
    await seedAuthenticatedCandidate(page);
    await installCandidateApiMocks(page, ({ pathname }) => {
      if (pathname === "/api/user/me") {
        return { data: {
          ...candidateUser,
          account_status: {
            certification: { available: true, is_candidate: scenario.purchaseCount > 0, purchase_count: scenario.purchaseCount, programs: [] },
            membership: { available: true, is_member: scenario.isMember },
          },
        } };
      }
      return emptyPortalResponse({ pathname } as ApiMockContext);
    });
    await page.goto("/dashboard", { waitUntil: "domcontentloaded" });
    await expect(page.getByRole("heading", { name: "欢迎回来，Regression" })).toBeVisible();
    await expect(page.getByTestId("dashboard-next-step")).toHaveText(new RegExp(scenario.label));
    await expect(page.getByTestId("dashboard-next-step")).toHaveAttribute("href", scenario.href);
  });
}

test("首页没有姓名时使用通用问候", async ({ page }) => {
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, ({ pathname }) => pathname === "/api/user/me"
    ? { data: { ...candidateUser, first_name: "", display_name: "", real_name: "" } }
    : emptyPortalResponse({ pathname } as ApiMockContext));
  await page.goto("/dashboard", { waitUntil: "domcontentloaded" });
  await expect(page.getByRole("heading", { name: "欢迎回来", exact: true })).toBeVisible();
});

test("公开商城分开展示课程认证与会员入口，并显示最低价", async ({ page }) => {
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
            category_tips: "Certification",
            display_amount_min: 12000,
            display_amount_max: 18000,
            display_currency: "USD",
          },
          {
            bundle_id: "public-course",
            pipeline_id: "public-course-pipeline",
            is_pipeline_bundle: true,
            name: "公开课程",
            category_tips: "Course",
            display_amount_min: 5000,
            display_amount_max: 5000,
            display_currency: "USD",
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
  const courseCard = page.locator("article.product-card").filter({ hasText: "公开课程" });
  const membershipCard = page.locator("article.product-card").filter({ hasText: "公开会员服务" });
  await expect(page.getByRole("button", { name: "全部", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "课程", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "会员方案" })).toHaveAttribute("href", "/membership");
  await expect(certificationCard.getByRole("button")).toContainText("去报名");
  await expect(certificationCard.locator(".product-price strong")).toContainText("起价");
  await expect(certificationCard.locator(".product-price strong")).toContainText("120");
  await expect(courseCard.locator(".product-price strong")).toContainText("50");
  await expect(courseCard.locator(".product-price strong")).not.toContainText("起价");
  await expect(membershipCard).toHaveCount(0);
  await page.getByRole("button", { name: "课程", exact: true }).click();
  await expect(certificationCard).toHaveCount(0);
  await expect(courseCard).toBeVisible();
  await page.getByRole("button", { name: "课程", exact: true }).click();
  await expect(certificationCard).toBeVisible();

  await page.evaluate(() => {
    localStorage.setItem("app_lang", "en");
    window.dispatchEvent(new Event("lang_change"));
  });
  await expect(page.getByRole("button", { name: "Course", exact: true })).toBeVisible();
  await expect(certificationCard.getByRole("button")).toContainText("Enroll Now");
  await expect(certificationCard.locator(".product-price strong")).toContainText("From");
  await expect(page.getByRole("link", { name: "Memberships" })).toHaveAttribute("href", "/membership");

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByRole("button", { name: "Certification", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Memberships" })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false);
});

test("登录后认证目录和会员购买入口分开，购买状态保持可用", async ({ page }) => {
  await seedAuthenticatedCandidate(page);
  let releaseEligibilityCheck: (() => void) | undefined;
  const eligibilityCheckPending = new Promise<void>((resolve) => {
    releaseEligibilityCheck = resolve;
  });
  await installCandidateApiMocks(page, async ({ pathname }) => {
    if (pathname === "/api/mall/bundles/candidate-certification-loading") {
      await eligibilityCheckPending;
      return {
        data: {
          bundle_id: "candidate-certification-loading",
          pipeline_id: "candidate-pipeline-loading",
          is_pipeline_bundle: true,
          name: "资格确认中认证项目",
          eligibility: {
            can_purchase: false,
            blockers: [{ blocker_type: "FORBIDDEN_QUALIFICATION" }],
          },
        },
      };
    }
    if (pathname === "/api/mall/bundles/candidate-certification-failed") {
      return { status: 500, errorCode: "ELIGIBILITY_CHECK_FAILED" };
    }
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
          {
            bundle_id: "candidate-certification-pending",
            pipeline_id: "candidate-pipeline-pending",
            is_pipeline_bundle: true,
            name: "待确认认证项目",
          },
          {
            bundle_id: "candidate-membership-pending",
            membership_id: "candidate-membership-pending",
            is_membership_bundle: true,
            name: "待确认会员服务",
          },
          {
            bundle_id: "candidate-certification-loading",
            pipeline_id: "candidate-pipeline-loading",
            is_pipeline_bundle: true,
            name: "资格确认中认证项目",
          },
          {
            bundle_id: "candidate-certification-failed",
            pipeline_id: "candidate-pipeline-failed",
            is_pipeline_bundle: true,
            name: "资格确认失败认证项目",
          },
        ],
      },
    };
  });

  await page.goto("/certifications", { waitUntil: "domcontentloaded" });

  await expect(page.getByRole("button", { name: "课程", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "会员方案" })).toHaveAttribute("href", "/membership");
  const certificationCard = page.locator('[data-bundle-id="candidate-certification"]');
  const membershipCard = page.locator('[data-bundle-id="candidate-membership"]');
  const pendingCertificationCard = page.locator('[data-bundle-id="candidate-certification-pending"]');
  const pendingMembershipCard = page.locator('[data-bundle-id="candidate-membership-pending"]');
  const loadingCertificationCard = page.locator('[data-bundle-id="candidate-certification-loading"]');
  const failedCertificationCard = page.locator('[data-bundle-id="candidate-certification-failed"]');
  await expect(certificationCard).toContainText("可报名专业认证考试");
  await expect(certificationCard).toContainText("去报名");
  await expect(certificationCard.locator(".course-card-stats")).toBeVisible();
  await expect(membershipCard).toHaveCount(0);
  await expect(pendingCertificationCard).toContainText("待确认报考资格");
  await expect(pendingCertificationCard).toContainText("查看报考资格");

  await loadingCertificationCard.click();
  await expect(loadingCertificationCard).toHaveAttribute("aria-busy", "true");
  await expect(loadingCertificationCard).toContainText("正在确认报考资格...");
  await expect(loadingCertificationCard).toContainText("请稍候...");
  releaseEligibilityCheck?.();
  await expect(loadingCertificationCard).toContainText("暂不可报名");

  await failedCertificationCard.click();
  await expect(failedCertificationCard).toContainText("报考资格获取失败");
  await expect(failedCertificationCard).toContainText("重新检查报考资格");
  await expect(page).toHaveURL(/\/certifications$/);

  await page.evaluate(() => {
    localStorage.setItem("app_lang", "en");
    window.dispatchEvent(new Event("lang_change"));
  });
  await expect(page.getByRole("button", { name: "Course", exact: true })).toBeVisible();
  await expect(certificationCard).toContainText("Ready to enroll in a certification exam");
  await expect(certificationCard).toContainText("Enroll Now");
  await expect(pendingCertificationCard).toContainText("Enrollment eligibility pending");
  await expect(pendingCertificationCard).toContainText("Check Enrollment Eligibility");
  await expect(loadingCertificationCard).toContainText("Enrollment unavailable");
  await expect(failedCertificationCard).toContainText("Failed to load enrollment eligibility");
  await expect(failedCertificationCard).toContainText("Retry Enrollment Check");

  await page.goto("/certifications?category=membership", { waitUntil: "domcontentloaded" });
  await expect(page.getByRole("heading", { name: "会员方案商品" })).toBeVisible();
  await expect(certificationCard).toHaveCount(0);
  await page.evaluate(() => {
    localStorage.setItem("app_lang", "en");
    window.dispatchEvent(new Event("lang_change"));
  });
  await expect(page.getByRole("heading", { name: "Membership offers" })).toBeVisible();
  await expect(membershipCard).toContainText("Eligible for membership");
  await expect(membershipCard).toContainText("Become a Member");
  await expect(membershipCard.locator(".course-card-stats")).toHaveCount(0);
  await expect(pendingMembershipCard).toContainText("Membership eligibility pending");
  await expect(pendingMembershipCard).toContainText("Check Membership Eligibility");
});

test("会员页购买按钮进入会员方案商品", async ({ page }) => {
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/membership/plans") {
      return { data: { plans: [{ membership_ulid: "plan-1", name: "CFtP Member", tier_level: 1 }] } };
    }
    if (pathname === "/api/mall/bundles") {
      return { data: { bundles: [{
        bundle_id: "membership-offer",
        membership_id: "plan-1",
        is_membership_bundle: true,
        name: "CFtP Member 方案",
        eligibility: { can_purchase: true, blockers: [] },
      }] } };
    }
    return emptyPortalResponse({ pathname } as ApiMockContext);
  });

  await page.goto("/membership", { waitUntil: "domcontentloaded" });
  await page.getByRole("button", { name: "会员等级" }).click();
  await page.getByRole("link", { name: "查看会员方案商品" }).click();
  await expect(page).toHaveURL(/\/certifications\?category=membership$/);
  await expect(page.locator('[data-bundle-id="membership-offer"]')).toContainText("成为会员");
});

test("首页仅在接口提供进度时显示进度条", async ({ page }) => {
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/pipeline") return { data: { list: [
      { pipeline_ulid: "owned-2", pipeline_cc_ulid: "config-2", pipeline_name: "CFtA", status: "PIPELINE_STATUS_RUNNING", progress: 0, progress_available: false },
    ] } };
    return emptyPortalResponse({ pathname } as ApiMockContext);
  });
  await page.goto("/dashboard", { waitUntil: "domcontentloaded" });
  await expect(page.getByTestId("dashboard-learning-card")).toBeVisible();
  await expect(page.getByTestId("dashboard-learning-card")).not.toContainText("0%");
});

test("支付成功页在移动端不会因长 ID 横向溢出", async ({ page }) => {
  const longOrderId = "ORDER-MOBILE-REGRESSION-0123456789-ABCDEFGHIJKLMNOPQRSTUVWXYZ-0123456789";

  await page.setViewportSize({ width: 382, height: 739 });
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, (context) => {
    if (context.pathname === `/api/orders/${longOrderId}`) {
      return { data: { items: [{ item_type: "pipeline", item_id: "pipeline-mobile" }] } };
    }
    return emptyPortalResponse(context);
  });
  await page.goto(`/checkout/success/${longOrderId}`, { waitUntil: "domcontentloaded" });

  const card = page.locator(".checkout-success-card");
  const orderId = page.locator(".checkout-success-id").first();

  await expect(card).toBeVisible();
  await expect(page.getByRole("link", { name: "查看我的专业认证" })).toBeVisible();
  await expect(page.getByRole("link", { name: "返回认证与课程" })).toBeVisible();
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await expect.poll(() => orderId.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(true);

  const cardBox = await card.boundingBox();
  expect(cardBox).not.toBeNull();
  expect(cardBox!.width).toBeLessThanOrEqual(350);
  await expect.poll(() => card.evaluate((element) => getComputedStyle(element).paddingLeft)).toBe("20px");
});

test("会员购买成功页进入会员中心而不是我的认证", async ({ page }) => {
  await seedAuthenticatedCandidate(page);
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/orders/membership-order") {
      return {
        data: {
          items: [{ item_type: "membership", item_id: "membership-plan-1" }],
        },
      };
    }
    return undefined;
  });

  await page.goto("/checkout/success/membership-order", { waitUntil: "domcontentloaded" });

  await expect(page.getByRole("heading", { name: "会员资格购买成功" })).toBeVisible();
  await expect(page.getByText("您的会员资格订单已成功提交。", { exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "查看我的会员资格" })).toHaveAttribute("href", "/membership");
  await expect(page.getByRole("link", { name: "查看我的专业认证" })).toHaveCount(0);
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
  await expect(dialog.getByRole("heading", { name: "工作经验证明模板", exact: true })).toBeVisible()
  await expect(
    dialog.locator(".credential-attachment-details").getByText("工作经验证明模板", { exact: true }),
  ).toBeVisible()
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
