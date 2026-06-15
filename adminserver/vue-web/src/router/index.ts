import { createRouter, createWebHistory } from "vue-router"
import CallbackPage from "@/pages/CallbackPage.vue"
import HomePage from "@/pages/HomePage.vue"
import LoginPage from "@/pages/LoginPage.vue"
import ResourcePacksPage from "@/pages/ResourcePacksPage.vue"
import SettingsPage from "@/pages/SettingsPage.vue"
import WhiteboxPage from "@/pages/WhiteboxPage.vue"

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", component: HomePage },
    { path: "/login", component: LoginPage },
    { path: "/callback", component: CallbackPage },
    { path: "/resource-packs", component: ResourcePacksPage },
    { path: "/whitebox", component: WhiteboxPage },
    { path: "/settings", component: SettingsPage },
  ],
})
