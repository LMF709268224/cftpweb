import { createRouter, createWebHistory } from "vue-router"
import { isAuthenticated, rememberPostLoginRedirect } from "@/lib/authStorage"
import { useUser } from "@/lib/user"

const GfiLandingPage = () => import("@/pages/GfiLandingPage.vue")
const GfiAboutPages = () => import("@/pages/GfiAboutPages.vue")
const GfiOfficialAboutPage = () => import("@/pages/GfiOfficialAboutPage.vue")
const GfiCertificationPage = () => import("@/pages/GfiCertificationPage.vue")
const GfiEcosystemPage = () => import("@/pages/GfiEcosystemPage.vue")
const GfiPublicationsPage = () => import("@/pages/GfiPublicationsPage.vue")
const GfiEventsPage = () => import("@/pages/GfiEventsPage.vue")
const GfiContentPage = () => import("@/pages/GfiContentPage.vue")
const PublicMarketplacePage = () => import("@/pages/PublicMarketplacePage.vue")
const LoginPage = () => import("@/pages/LoginPage.vue")
const CallbackPage = () => import("@/pages/CallbackPage.vue")
const CoursesPage = () => import("@/pages/CoursesPage.vue")
const MyCertificationsPage = () => import("@/pages/MyCertificationsPage.vue")
const CourseDetailPage = () => import("@/pages/CourseDetailPage.vue")
const CourseLearnPage = () => import("@/pages/CourseLearnPage.vue")
const CourseSupplementaryPage = () => import("@/pages/CourseSupplementaryPage.vue")
const CourseTimelinePage = () => import("@/pages/CourseTimelinePage.vue")
const MembershipPage = () => import("@/pages/MembershipPage.vue")
const ExamsPage = () => import("@/pages/ExamsPage.vue")
const ExamResultPage = () => import("@/pages/ExamResultPage.vue")
const ExamSignupPage = () => import("@/pages/ExamSignupPage.vue")
const RecordsPage = () => import("@/pages/RecordsPage.vue")
const CredentialsPage = () => import("@/pages/CredentialsPage.vue")
const CertificatesPage = () => import("@/pages/CertificatesPage.vue")
const OrdersPage = () => import("@/pages/OrdersPage.vue")
const MessagesPage = () => import("@/pages/MessagesPage.vue")
const SettingsPage = () => import("@/pages/SettingsPage.vue")
const QuizPage = () => import("@/pages/QuizPage.vue")
const ResourcePackDetailPage = () => import("@/pages/ResourcePackDetailPage.vue")
const ResourcePacksPage = () => import("@/pages/ResourcePacksPage.vue")
const PdfPreviewPage = () => import("@/pages/PdfPreviewPage.vue")
const VideoPreviewPage = () => import("@/pages/VideoPreviewPage.vue")
const InvoiceRedirectPage = () => import("@/pages/InvoiceRedirectPage.vue")
const PaymentBridgePage = () => import("@/pages/PaymentBridgePage.vue")
const CheckoutWizardPage = () => import("@/pages/CheckoutWizardPage.vue")
const CheckoutSuccessPage = () => import("@/pages/CheckoutSuccessPage.vue")
const NotFoundPage = () => import("@/pages/NotFoundPage.vue")

function firstRouteValue(value: unknown) {
  if (Array.isArray(value)) return String(value[0] || "")
  return String(value || "")
}

function redirectToCertifications(to: any) {
  return { path: "/certifications", query: to.query, hash: to.hash }
}

function redirectCertificationDetail(to: any) {
  const pipelineId = firstRouteValue(to.query.id || to.params.pipelineId)
  return pipelineId ? { path: `/certifications/${encodeURIComponent(pipelineId)}`, hash: to.hash } : redirectToCertifications(to)
}

function redirectCertificationLearn(to: any) {
  const pipelineId = firstRouteValue(to.query.pipelineId || to.params.pipelineId)
  const courseId = firstRouteValue(to.query.courseId || to.params.courseId)
  const lessonId = firstRouteValue(to.query.lessonId || to.params.lessonId)
  if (!pipelineId || !courseId) return redirectToCertifications(to)
  const base = `/certifications/${encodeURIComponent(pipelineId)}/learn/${encodeURIComponent(courseId)}`
  return { path: lessonId ? `${base}/lessons/${encodeURIComponent(lessonId)}` : base, hash: to.hash }
}

function redirectCertificationSupplementary(to: any) {
  const pipelineId = firstRouteValue(to.query.pipelineId || to.params.pipelineId)
  const courseId = firstRouteValue(to.query.courseId || to.params.courseId)
  return pipelineId && courseId
    ? { path: `/certifications/${encodeURIComponent(pipelineId)}/supplementary/${encodeURIComponent(courseId)}`, hash: to.hash }
    : redirectToCertifications(to)
}

function redirectCertificationTimeline(to: any) {
  const pipelineId = firstRouteValue(to.query.id || to.params.pipelineId)
  return pipelineId ? { path: `/certifications/${encodeURIComponent(pipelineId)}/timeline`, hash: to.hash } : redirectToCertifications(to)
}

function redirectResourcePackDetail(to: any) {
  const packId = firstRouteValue(to.query.id || to.params.packId)
  return packId ? { path: `/resource-packs/${encodeURIComponent(packId)}`, hash: to.hash } : { path: "/resource-packs", query: to.query, hash: to.hash }
}

export const router = createRouter({
  history: createWebHistory(),
  scrollBehavior(to, _from, savedPosition) {
    if (savedPosition) return savedPosition
    if (to.hash) return { el: to.hash }
    return { left: 0, top: 0 }
  },
  routes: [
    { path: "/", component: GfiLandingPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi", redirect: "/" },
    { path: "/gfi/about", component: GfiAboutPages, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/about/board-of-directors", component: GfiOfficialAboutPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/about/team", component: GfiOfficialAboutPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/subcommittees", component: GfiOfficialAboutPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/industry-fellow", component: GfiOfficialAboutPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/youth-wing", component: GfiOfficialAboutPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/career", component: GfiOfficialAboutPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/contact", component: GfiOfficialAboutPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/certifications", component: GfiCertificationPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/certifications/pathway", component: GfiCertificationPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/programmes/cfta", component: GfiCertificationPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/programmes/cftp", component: GfiCertificationPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/programmes", component: GfiEcosystemPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/partnerships", component: GfiEcosystemPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/membership/benefits", component: GfiEcosystemPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/membership/corporate", component: GfiEcosystemPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/publications/reports", component: GfiPublicationsPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/publications/insights", component: GfiPublicationsPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/publications/news", component: GfiPublicationsPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/publications/journals", component: GfiPublicationsPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/events/all-events", component: GfiEventsPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/events/webinar-recordings", component: GfiEventsPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/events/conferences", component: GfiEventsPage, meta: { titleKey: "gfiLanding" } },
    { path: "/gfi/:pathMatch(.*)+", component: GfiContentPage, meta: { titleKey: "gfiLanding" } },
    { path: "/marketplace", component: PublicMarketplacePage, meta: { titleKey: "marketplace" } },
    { path: "/login", component: LoginPage, meta: { titleKey: "login" } },
    { path: "/callback", component: CallbackPage, meta: { titleKey: "callback" } },
    { path: "/certifications", component: CoursesPage, meta: { titleKey: "marketplace" } },
    { path: "/my-certifications", component: MyCertificationsPage, meta: { titleKey: "myCertifications", requiresAuth: true } },
    { path: "/certifications/detail", redirect: redirectCertificationDetail },
    { path: "/certifications/learn", redirect: redirectCertificationLearn },
    { path: "/certifications/supplementary", redirect: redirectCertificationSupplementary },
    { path: "/certifications/timeline", redirect: redirectCertificationTimeline },
    { path: "/certifications/:pipelineId", component: CourseDetailPage, meta: { titleKey: "certificationDetail" } },
    { path: "/certifications/:pipelineId/learn/:courseId", component: CourseLearnPage, meta: { titleKey: "courseLearning", requiresAuth: true } },
    { path: "/certifications/:pipelineId/learn/:courseId/lessons/:lessonId", component: CourseLearnPage, meta: { titleKey: "courseLearning", requiresAuth: true } },
    { path: "/certifications/:pipelineId/supplementary/:courseId", component: CourseSupplementaryPage, meta: { titleKey: "supplementaryMaterials", requiresAuth: true } },
    { path: "/certifications/:pipelineId/timeline", component: CourseTimelinePage, meta: { titleKey: "timeline", requiresAuth: true } },
    { path: "/courses", redirect: redirectToCertifications },
    { path: "/courses/detail", redirect: redirectCertificationDetail },
    { path: "/courses/learn", redirect: redirectCertificationLearn },
    { path: "/courses/supplementary", redirect: redirectCertificationSupplementary },
    { path: "/courses/timeline", redirect: redirectCertificationTimeline },
    { path: "/membership", component: MembershipPage, meta: { titleKey: "membership" } },
    { path: "/exams", component: ExamsPage, meta: { titleKey: "exams", requiresAuth: true } },
    { path: "/checkout/success/:orderId", component: CheckoutSuccessPage, meta: { titleKey: "checkoutSuccess", requiresAuth: true } },
    { path: "/checkout/:bundleId", component: CheckoutWizardPage, meta: { titleKey: "checkout", requiresAuth: true } },
    { path: "/exams/result", component: ExamResultPage, meta: { titleKey: "examResult", requiresAuth: true } },
    { path: "/exams/signup", component: ExamSignupPage, meta: { titleKey: "examSignup", requiresAuth: true } },
    { path: "/records", component: RecordsPage, meta: { titleKey: "records", requiresAuth: true } },
    { path: "/resource-packs", component: ResourcePacksPage, meta: { titleKey: "resourcePacks", requiresAuth: true } },
    { path: "/resource-packs/detail", redirect: redirectResourcePackDetail },
    { path: "/resource-packs/:packId", component: ResourcePackDetailPage, meta: { titleKey: "resourcePackDetail", requiresAuth: true } },
    { path: "/resource-pack-files/:fileId/preview", component: PdfPreviewPage, meta: { titleKey: "pdfPreview", requiresAuth: true } },
    { path: "/video-preview/resource-pack-files/:fileId", component: VideoPreviewPage, meta: { titleKey: "videoPreview", requiresAuth: true } },
    { path: "/credentials", component: CredentialsPage, meta: { titleKey: "credentials", requiresAuth: true } },
    { path: "/certificates", component: CertificatesPage, meta: { titleKey: "certificates", requiresAuth: true } },
    { path: "/orders", component: OrdersPage, meta: { titleKey: "orders", requiresAuth: true } },
    { path: "/messages", component: MessagesPage, meta: { titleKey: "messages", requiresAuth: true } },
    { path: "/settings", component: SettingsPage, meta: { titleKey: "settings", requiresAuth: true } },
    { path: "/quizzes", component: QuizPage, meta: { titleKey: "quiz", requiresAuth: true } },
    { path: "/pdf-preview/lessons/:lessonId", component: PdfPreviewPage, meta: { titleKey: "pdfPreview", requiresAuth: true } },
    { path: "/pdf-preview/resources/:resourceKey", component: PdfPreviewPage, meta: { titleKey: "pdfPreview", requiresAuth: true } },
    { path: "/pdf-preview", component: PdfPreviewPage, meta: { titleKey: "pdfPreview", requiresAuth: true } },
    { path: "/invoice-redirect", component: InvoiceRedirectPage, meta: { titleKey: "invoiceRedirect", requiresAuth: true } },
    { path: "/payment-bridge", component: PaymentBridgePage, meta: { titleKey: "paymentBridge", requiresAuth: true } },
    { path: "/:pathMatch(.*)*", component: NotFoundPage, meta: { titleKey: "notFound" } },
  ],
})

router.beforeEach((to) => {
  if (to.meta.requiresAuth && !isAuthenticated()) {
    rememberPostLoginRedirect(to.fullPath)
    return { path: "/login", replace: true }
  }

  if (to.path === "/exams/signup" && isAuthenticated()) {
    void useUser().fetchUser()
  }
})
