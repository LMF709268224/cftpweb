import { createApp } from "vue"
import { Toaster } from "vue-sonner"
import App from "./App.vue"
import { modalDialog } from "./lib/modalDialog"
import router from "./router"
import "./styles/globals.css"
import "vue-sonner/style.css"

createApp(App).component("Toaster", Toaster).directive("modal-dialog", modalDialog).use(router).mount("#app")
