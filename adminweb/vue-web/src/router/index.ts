import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router"
import { isAuthenticated } from "@/lib/authStorage"

const AdminLayout = () => import("@/components/AdminLayout.vue")
const AdminOpsPage = () => import("@/pages/AdminOpsPage.vue")
const AuditLogsPage = () => import("@/pages/AuditLogsPage.vue")
const ApplicationsPage = () => import("@/pages/ApplicationsPage.vue")
const BundlesPage = () => import("@/pages/BundlesPage.vue")
const CallbackPage = () => import("@/pages/CallbackPage.vue")
const CredentialsPage = () => import("@/pages/CredentialsPage.vue")
const DashboardPage = () => import("@/pages/DashboardPage.vue")
const ExamsPage = () => import("@/pages/ExamsPage.vue")
const InvoicesPage = () => import("@/pages/InvoicesPage.vue")
const LmsPage = () => import("@/pages/LmsPage.vue")
const LoginPage = () => import("@/pages/LoginPage.vue")
const MailsPage = () => import("@/pages/MailsPage.vue")
const MessagesPage = () => import("@/pages/MessagesPage.vue")
const OrdersPage = () => import("@/pages/OrdersPage.vue")
const PdfRequestsPage = () => import("@/pages/PdfRequestsPage.vue")
const PdfTemplatesPage = () => import("@/pages/PdfTemplatesPage.vue")
const PermissionsPage = () => import("@/pages/PermissionsPage.vue")
const PipelinesPage = () => import("@/pages/PipelinesPage.vue")
const ProgPage = () => import("@/pages/ProgPage.vue")
const ResourcePackFilesPage = () => import("@/pages/ResourcePackFilesPage.vue")
const ResourcePacksPage = () => import("@/pages/ResourcePacksPage.vue")
const SettingsPage = () => import("@/pages/SettingsPage.vue")
const WebhookAuditPage = () => import("@/pages/WebhookAuditPage.vue")

export type ResourceRouteMeta = {
  copyKey: ResourceRouteKey
  endpoint: string
  itemKeys: string[]
  pagination?: "offset" | "page"
}

export type ResourceRouteKey =
  | "dashboard"
  | "resourcePacks"
  | "resourcePackFiles"
  | "lms"
  | "pipelines"
  | "bundles"
  | "prog"
  | "exams"
  | "messages"
  | "mails"
  | "orders"
  | "invoices"
  | "credentials"
  | "applications"
  | "pdfTemplates"
  | "pdfRequests"
  | "auditLogs"
  | "adminOps"
  | "webhooks"
  | "permissions"
  | "settings"

export const resourceRoutes: RouteRecordRaw[] = [
  {
    path: "/dashboard",
    name: "dashboard",
    component: DashboardPage,
    meta: {
      copyKey: "dashboard",
      endpoint: "/api/dashboard/ops",
      itemKeys: [],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/resource-packs",
    name: "resource-packs",
    component: ResourcePacksPage,
    meta: {
      copyKey: "resourcePacks",
      endpoint: "/api/lms/resource-packs",
      itemKeys: ["packs", "items"],
      pagination: "page",
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/resource-pack-files",
    name: "resource-pack-files",
    component: ResourcePackFilesPage,
    meta: {
      copyKey: "resourcePackFiles",
      endpoint: "/api/lms/resource-pack-files",
      itemKeys: ["files", "items"],
      pagination: "page",
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/lms",
    name: "lms",
    component: LmsPage,
    meta: {
      copyKey: "lms",
      endpoint: "/api/lms/courses",
      itemKeys: ["courses", "items"],
      pagination: "page",
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/pipelines",
    name: "pipelines",
    component: PipelinesPage,
    meta: {
      copyKey: "pipelines",
      endpoint: "/api/pipelines",
      itemKeys: ["pipelines", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/bundles",
    name: "bundles",
    component: BundlesPage,
    meta: {
      copyKey: "bundles",
      endpoint: "/api/mall/bundles",
      itemKeys: ["bundles", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/prog",
    name: "prog",
    component: ProgPage,
    meta: {
      copyKey: "prog",
      endpoint: "/api/prog/pipelines",
      itemKeys: ["pipelines", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/exams",
    name: "exams",
    component: ExamsPage,
    meta: {
      copyKey: "exams",
      endpoint: "/api/exams",
      itemKeys: ["exams", "items"],
      pagination: "page",
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/messages",
    name: "messages",
    component: MessagesPage,
    meta: {
      copyKey: "messages",
      endpoint: "/api/messages",
      itemKeys: ["messages", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/mails",
    name: "mails",
    component: MailsPage,
    meta: {
      copyKey: "mails",
      endpoint: "/api/mails",
      itemKeys: ["mails", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/orders",
    name: "orders",
    component: OrdersPage,
    meta: {
      copyKey: "orders",
      endpoint: "/api/mall/orders",
      itemKeys: ["orders", "items"],
      pagination: "page",
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/invoices",
    name: "invoices",
    component: InvoicesPage,
    meta: {
      copyKey: "invoices",
      endpoint: "/api/invoices",
      itemKeys: ["invoices", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/credentials",
    name: "credentials",
    component: CredentialsPage,
    meta: {
      copyKey: "credentials",
      endpoint: "/api/credentials/definitions",
      itemKeys: ["definitions", "credentials", "items"],
      pagination: "page",
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/applications",
    name: "applications",
    component: ApplicationsPage,
    meta: {
      copyKey: "applications",
      endpoint: "/api/credentials/applications",
      itemKeys: ["applications", "items"],
      pagination: "page",
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/pdf-templates",
    name: "pdf-templates",
    component: PdfTemplatesPage,
    meta: {
      copyKey: "pdfTemplates",
      endpoint: "/api/pdf-templates",
      itemKeys: ["templates", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/pdf-requests",
    name: "pdf-requests",
    component: PdfRequestsPage,
    meta: {
      copyKey: "pdfRequests",
      endpoint: "/api/pdf-requests",
      itemKeys: ["requests", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/admin-ops",
    name: "admin-ops",
    component: AdminOpsPage,
    meta: {
      copyKey: "adminOps",
      endpoint: "/api/pay/webhook-events",
      itemKeys: ["items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/audit/logs",
    name: "audit-logs",
    component: AuditLogsPage,
    meta: {
      copyKey: "auditLogs",
      endpoint: "/api/audit/logs",
      itemKeys: ["items"],
      pagination: "page",
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/audit/webhooks",
    name: "audit-webhooks",
    component: WebhookAuditPage,
    meta: {
      copyKey: "webhooks",
      endpoint: "/api/audit/webhooks",
      itemKeys: ["webhooks", "events", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/permissions",
    name: "permissions",
    component: PermissionsPage,
    meta: {
      copyKey: "permissions",
      endpoint: "/api/permissions",
      itemKeys: ["permissions", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/settings",
    name: "settings",
    component: SettingsPage,
    meta: {
      copyKey: "settings",
      endpoint: "/api/user/me",
      itemKeys: [],
    } satisfies ResourceRouteMeta,
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/login", name: "login", component: LoginPage },
    { path: "/callback", name: "callback", component: CallbackPage },
    {
      path: "/",
      component: AdminLayout,
      children: [{ path: "", redirect: "/dashboard" }, ...resourceRoutes],
    },
  ],
})

router.beforeEach((to) => {
  if (to.name === "login" || to.name === "callback") {
    return true
  }

  if (!isAuthenticated()) {
    return { name: "login" }
  }

  return true
})

export default router
