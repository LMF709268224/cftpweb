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

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", component: HomePage },
    { path: "/login", component: LoginPage },
    { path: "/callback", component: CallbackPage },
    { path: "/courses", component: CoursesPage },
    { path: "/courses/detail", component: CourseDetailPage },
    { path: "/courses/learn", component: CourseLearnPage },
    { path: "/courses/supplementary", component: CourseSupplementaryPage },
    { path: "/courses/timeline", component: CourseTimelinePage },
    { path: "/membership", component: MembershipPage },
    { path: "/exams", component: ExamsPage },
    { path: "/exams/result", component: ExamResultPage },
    { path: "/exams/signup", component: ExamSignupPage },
    { path: "/records", component: RecordsPage },
    { path: "/resource-packs", component: ResourcePacksPage },
    { path: "/resource-packs/detail", component: ResourcePackDetailPage },
    { path: "/resource-packs/:packId", component: ResourcePackDetailPage },
    { path: "/resource-pack-files/:fileId/preview", component: PdfPreviewPage },
    { path: "/credentials", component: CredentialsPage },
    { path: "/certificates", component: CertificatesPage },
    { path: "/orders", component: OrdersPage },
    { path: "/messages", component: MessagesPage },
    { path: "/settings", component: SettingsPage },
    { path: "/quizzes", component: QuizPage },
    { path: "/pdf-preview", component: PdfPreviewPage },
    { path: "/invoice-redirect", component: InvoiceRedirectPage },
  ],
})
