import { type App, type Plugin } from "vue"
import type { Router } from "vue-router"
import { telemetry } from "../lib/telemetry"

export const telemetryPlugin: Plugin = {
  install(app: App, options: { router?: Router } = {}) {
    // 1. Global Error Handling
    app.config.errorHandler = (err, _instance, info) => {
      telemetry.track("js_error", {
        message: err instanceof Error ? err.message : String(err),
        stack: err instanceof Error ? err.stack : undefined,
        info,
      })
      console.error("[Telemetry Caught]", err, info)
    }

    window.addEventListener("unhandledrejection", (event) => {
      telemetry.track("unhandled_rejection", {
        reason: event.reason instanceof Error ? event.reason.message : String(event.reason),
      })
    })

    // 2. Vue Router Navigation Tracking
    if (options.router) {
      options.router.afterEach((to, from) => {
        telemetry.track("page_view", {
          path: to.path,
          from_path: from.path,
          name: to.name?.toString(),
        })
      })
    }

    // 3. Custom v-track directive
    // Usage: <button v-track:click="'submit_exam'">Submit</button>
    // Usage with payload: <button v-track:click="{ event: 'view_item', payload: { id: 1 } }">View</button>
    app.directive("track", {
      mounted(el: HTMLElement, binding) {
        const eventType = binding.arg || "click"
        
        const handler = () => {
          let eventName = ""
          let payload: Record<string, unknown> | undefined = undefined

          if (typeof binding.value === "string") {
            eventName = binding.value
          } else if (typeof binding.value === "object" && binding.value !== null) {
            eventName = binding.value.event || "unknown_event"
            payload = binding.value.payload
          }

          if (eventName) {
            telemetry.track(eventName, payload)
          }
        }
        
        el.addEventListener(eventType, handler)
        // Store handler for cleanup
        ;(el as any).__telemetry_handler__ = { eventType, handler }
      },
      unmounted(el: HTMLElement) {
        const data = (el as any).__telemetry_handler__
        if (data) {
          el.removeEventListener(data.eventType, data.handler)
        }
      }
    })
  }
}
