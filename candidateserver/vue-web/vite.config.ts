import { fileURLToPath, URL } from "node:url"
import tailwindcss from "@tailwindcss/vite"
import vue from "@vitejs/plugin-vue"
import { defineConfig } from "vite"

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
      "@embedpdf/vue-pdf-viewer": fileURLToPath(new URL("./node_modules/@embedpdf/vue-pdf-viewer/dist/index.js", import.meta.url)),
      "@embedpdf/snippet": fileURLToPath(new URL("./node_modules/@embedpdf/snippet/dist/embedpdf.js", import.meta.url)),
      "@vue/reactivity": fileURLToPath(new URL("./node_modules/@vue/reactivity/dist/reactivity.esm-bundler.js", import.meta.url)),
      "@vue/runtime-core": fileURLToPath(new URL("./node_modules/@vue/runtime-core/dist/runtime-core.esm-bundler.js", import.meta.url)),
      "@vue/runtime-dom": fileURLToPath(new URL("./node_modules/@vue/runtime-dom/dist/runtime-dom.esm-bundler.js", import.meta.url)),
      "@vue/shared": fileURLToPath(new URL("./node_modules/@vue/shared/dist/shared.esm-bundler.js", import.meta.url)),
      "@vue/devtools-api": fileURLToPath(new URL("./node_modules/@vue/devtools-api/dist/index.js", import.meta.url)),
      "@vue/devtools-kit": fileURLToPath(new URL("./node_modules/@vue/devtools-kit/dist/index.js", import.meta.url)),
      "@vue/devtools-shared": fileURLToPath(new URL("./node_modules/@vue/devtools-shared/dist/index.js", import.meta.url)),
    },
  },
  build: {
    minify: "esbuild",
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks(id) {
          const normalizedId = id.replace(/\\/g, "/")
          if (normalizedId.includes("/node_modules/vue/") || normalizedId.includes("/node_modules/vue-router/") || normalizedId.includes("/node_modules/@vue/")) return "vue"
          if (normalizedId.includes("lucide-vue-next")) return "icons"
          if (normalizedId.includes("vue-sonner")) return "notifications"
          return undefined
        },
      },
    },
  },
  server: {
    host: "0.0.0.0",
    port: 8081,
    proxy: {
      "/api": {
        target: "https://cftpcand.llwan.top",
        changeOrigin: true,
        
      },
    },
  },
})
