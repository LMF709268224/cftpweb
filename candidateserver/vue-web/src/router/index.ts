import { createRouter, createWebHistory } from "vue-router"
import HomePage from "@/pages/HomePage.vue"
import LoginPage from "@/pages/LoginPage.vue"
import CallbackPage from "@/pages/CallbackPage.vue"
import CoursesPage from "@/pages/CoursesPage.vue"
import CourseDetailPage from "@/pages/CourseDetailPage.vue"
import CourseLearnPage from "@/pages/CourseLearnPage.vue"
import CourseSupplementaryPage from "@/pages/CourseSupplementaryPage.vue"
import CourseTimelinePage from "@/pages/CourseTimelinePage.vue"
import MembershipPage from "@/pages/MembershipPage.vue"
import ExamsPage from "@/pages/ExamsPage.vue"
import ExamResultPage from "@/pages/ExamResultPage.vue"
import ExamSignupPage from "@/pages/ExamSignupPage.vue"
import RecordsPage from "@/pages/RecordsPage.vue"
import CredentialsPage from "@/pages/CredentialsPage.vue"
import CertificatesPage from "@/pages/CertificatesPage.vue"
import OrdersPage from "@/pages/OrdersPage.vue"
import MessagesPage from "@/pages/MessagesPage.vue"
import SettingsPage from "@/pages/SettingsPage.vue"
import QuizPage from "@/pages/QuizPage.vue"
import ResourcePackDetailPage from "@/pages/ResourcePackDetailPage.vue"
import ResourcePacksPage from "@/pages/ResourcePacksPage.vue"
import PdfPreviewPage from "@/pages/PdfPreviewPage.vue"
import InvoiceRedirectPage from "@/pages/InvoiceRedirectPage.vue"

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
  ],
})
