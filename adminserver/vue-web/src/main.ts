import { createApp } from "vue"
import { Toaster } from "vue-sonner"
import "vue-sonner/style.css"
import App from "./App.vue"
import { router } from "./router"
import "./styles/globals.css"

createApp(App)
  .component("Toaster", Toaster)
  .use(router)
  .mount("#app")
