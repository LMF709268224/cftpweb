import { fileURLToPath, URL } from "node:url"
import path from "node:path"
import tailwindcss from "@tailwindcss/vite"
import vue from "@vitejs/plugin-vue"
import { defineConfig } from "vite"

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
      "@embedpdf/snippet": fileURLToPath(new URL("./node_modules/@embedpdf/snippet/dist/embedpdf.js", import.meta.url)),
      "@vue/reactivity": fileURLToPath(new URL("./node_modules/@vue/reactivity/dist/reactivity.esm-bundler.js", import.meta.url)),
      "@vue/runtime-core": fileURLToPath(new URL("./node_modules/@vue/runtime-core/dist/runtime-core.esm-bundler.js", import.meta.url)),
      "@vue/runtime-dom": fileURLToPath(new URL("./node_modules/@vue/runtime-dom/dist/runtime-dom.esm-bundler.js", import.meta.url)),
      "@vue/shared": fileURLToPath(new URL("./node_modules/@vue/shared/dist/shared.esm-bundler.js", import.meta.url)),
      "@vue/devtools-api": fileURLToPath(new URL("./node_modules/@vue/devtools-api/dist/index.js", import.meta.url)),
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
