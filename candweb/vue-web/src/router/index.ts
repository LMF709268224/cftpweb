import { createRouter, createWebHistory } from "vue-router"

const HomePage = () => import("@/pages/HomePage.vue")
const LoginPage = () => import("@/pages/LoginPage.vue")
const CallbackPage = () => import("@/pages/CallbackPage.vue")
const CoursesPage = () => import("@/pages/CoursesPage.vue")
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
  routes: [
    { path: "/", component: HomePage },
    { path: "/login", component: LoginPage },
    { path: "/callback", component: CallbackPage },
    { path: "/certifications", component: CoursesPage },
    { path: "/certifications/detail", redirect: redirectCertificationDetail },
    { path: "/certifications/learn", redirect: redirectCertificationLearn },
    { path: "/certifications/supplementary", redirect: redirectCertificationSupplementary },
    { path: "/certifications/timeline", redirect: redirectCertificationTimeline },
    { path: "/certifications/:pipelineId", component: CourseDetailPage },
    { path: "/certifications/:pipelineId/learn/:courseId", component: CourseLearnPage },
    { path: "/certifications/:pipelineId/learn/:courseId/lessons/:lessonId", component: CourseLearnPage },
    { path: "/certifications/:pipelineId/supplementary/:courseId", component: CourseSupplementaryPage },
    { path: "/certifications/:pipelineId/timeline", component: CourseTimelinePage },
    { path: "/courses", redirect: redirectToCertifications },
    { path: "/courses/detail", redirect: redirectCertificationDetail },
    { path: "/courses/learn", redirect: redirectCertificationLearn },
    { path: "/courses/supplementary", redirect: redirectCertificationSupplementary },
    { path: "/courses/timeline", redirect: redirectCertificationTimeline },
    { path: "/membership", component: MembershipPage },
    { path: "/exams", component: ExamsPage },
    { path: "/exams/result", component: ExamResultPage },
    { path: "/exams/signup", component: ExamSignupPage },
    { path: "/records", component: RecordsPage },
    { path: "/resource-packs", component: ResourcePacksPage },
    { path: "/resource-packs/detail", redirect: redirectResourcePackDetail },
    { path: "/resource-packs/:packId", component: ResourcePackDetailPage },
    { path: "/resource-pack-files/:fileId/preview", component: PdfPreviewPage },
    { path: "/video-preview/resource-pack-files/:fileId", component: VideoPreviewPage },
    { path: "/credentials", component: CredentialsPage },
    { path: "/certificates", component: CertificatesPage },
    { path: "/orders", component: OrdersPage },
    { path: "/messages", component: MessagesPage },
    { path: "/settings", component: SettingsPage },
    { path: "/quizzes", component: QuizPage },
    { path: "/pdf-preview/lessons/:lessonId", component: PdfPreviewPage },
    { path: "/pdf-preview/resources/:resourceKey", component: PdfPreviewPage },
    { path: "/pdf-preview", component: PdfPreviewPage },
    { path: "/invoice-redirect", component: InvoiceRedirectPage },
    { path: "/payment-bridge", component: PaymentBridgePage },
  ],
})
