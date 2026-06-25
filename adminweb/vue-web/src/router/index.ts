import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router"
import AdminLayout from "@/components/AdminLayout.vue"
import { getAccessToken } from "@/lib/authStorage"
import ApplicationsPage from "@/pages/ApplicationsPage.vue"
import BundlesPage from "@/pages/BundlesPage.vue"
import CallbackPage from "@/pages/CallbackPage.vue"
import CredentialsPage from "@/pages/CredentialsPage.vue"
import InvoicesPage from "@/pages/InvoicesPage.vue"
import LmsPage from "@/pages/LmsPage.vue"
import LoginPage from "@/pages/LoginPage.vue"
import MailsPage from "@/pages/MailsPage.vue"
import MessagesPage from "@/pages/MessagesPage.vue"
import OrdersPage from "@/pages/OrdersPage.vue"
import PdfRequestsPage from "@/pages/PdfRequestsPage.vue"
import PdfTemplatesPage from "@/pages/PdfTemplatesPage.vue"
import PermissionsPage from "@/pages/PermissionsPage.vue"
import PipelinesPage from "@/pages/PipelinesPage.vue"
import ProgPage from "@/pages/ProgPage.vue"
import SettingsPage from "@/pages/SettingsPage.vue"
import WebhookAuditPage from "@/pages/WebhookAuditPage.vue"

export type ResourceRouteMeta = {
  title: string
  subtitle: string
  endpoint: string
  itemKeys: string[]
  pagination?: "offset" | "page"
}

export const resourceRoutes: RouteRecordRaw[] = [
  {
    path: "/lms",
    name: "lms",
    component: LmsPage,
    meta: {
      title: "课程配置",
      subtitle: "维护 GLMS 课程内容、发布状态和基础资料",
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
      title: "管线配置",
      subtitle: "维护认证管线、阶段、课程单元、证书和资格要求",
      endpoint: "/api/pipelines",
      itemKeys: ["pipelines", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/bundles",
    name: "bundles",
    component: BundlesPage,
    meta: {
      title: "商品配置",
      subtitle: "维护 Bundle 商品、定价快照和支付入口配置",
      endpoint: "/api/mall/bundles",
      itemKeys: ["bundles", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/prog",
    name: "prog",
    component: ProgPage,
    meta: {
      title: "管线管理",
      subtitle: "查看考生正在运行的管线实例和状态流转",
      endpoint: "/api/prog/pipelines",
      itemKeys: ["pipelines", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/messages",
    name: "messages",
    component: MessagesPage,
    meta: {
      title: "站内信",
      subtitle: "查看和管理站内通知",
      endpoint: "/api/messages",
      itemKeys: ["messages", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/mails",
    name: "mails",
    component: MailsPage,
    meta: {
      title: "邮件中心",
      subtitle: "查看邮件模板和投递记录",
      endpoint: "/api/mails",
      itemKeys: ["mails", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/orders",
    name: "orders",
    component: OrdersPage,
    meta: {
      title: "订单管理",
      subtitle: "查看认证、管线、阶段、重考和资格申请订单",
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
      title: "发票管理",
      subtitle: "查看发票和支付凭证",
      endpoint: "/api/invoices",
      itemKeys: ["invoices", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/credentials",
    name: "credentials",
    component: CredentialsPage,
    meta: {
      title: "资格定义",
      subtitle: "维护认证资格、免考资格和最终证书定义",
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
      title: "审核中心",
      subtitle: "审核考生提交的资格申请",
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
      title: "PDF 模板配置",
      subtitle: "维护证书和证明文件模板",
      endpoint: "/api/pdf-templates",
      itemKeys: ["templates", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/pdf-requests",
    name: "pdf-requests",
    component: PdfRequestsPage,
    meta: {
      title: "证书生成流水",
      subtitle: "查看证书 PDF 生成任务",
      endpoint: "/api/pdf-requests",
      itemKeys: ["requests", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/audit/webhooks",
    name: "audit-webhooks",
    component: WebhookAuditPage,
    meta: {
      title: "Webhook 审计",
      subtitle: "查看支付和外部系统回调审计",
      endpoint: "/api/audit/webhooks",
      itemKeys: ["webhooks", "events", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/permissions",
    name: "permissions",
    component: PermissionsPage,
    meta: {
      title: "考生权限管理",
      subtitle: "查看和调整考生访问权限",
      endpoint: "/api/permissions",
      itemKeys: ["permissions", "items"],
    } satisfies ResourceRouteMeta,
  },
  {
    path: "/settings",
    name: "settings",
    component: SettingsPage,
    meta: {
      title: "账户设置",
      subtitle: "维护管理员个人资料和登录密码",
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
      children: [{ path: "", redirect: "/lms" }, ...resourceRoutes],
    },
  ],
})

router.beforeEach((to) => {
  if (to.name === "login" || to.name === "callback") {
    return true
  }

  if (!getAccessToken()) {
    return { name: "login" }
  }

  return true
})

export default router
