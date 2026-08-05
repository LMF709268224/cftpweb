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
