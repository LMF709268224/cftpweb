import { createApp } from "vue"
import { Toaster } from "vue-sonner"
import "vue-sonner/style.css"
import "./styles/globals.css"
import App from "./App.vue"
import { router } from "./router"

createApp(App)
  .component("Toaster", Toaster)
  .use(router)
  .mount("#app")
