import { defineConfig } from "vite";
import laravel from "laravel-vite-plugin";
{{- if .FrontendHasReact}}
import react from "@vitejs/plugin-react";
{{- end}}
{{- if .FrontendHasVue}}
import vue from "@vitejs/plugin-vue";
{{- end}}
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [
    tailwindcss(),
    laravel({
      {{- if .FrontendHasReact}}
      input: ["resources/js/app.tsx", "resources/css/app.css"],
      ssr: "resources/js/ssr.tsx",
      {{- end}}
      {{- if .FrontendHasVue}}
      input: ["resources/js/app.js", "resources/css/app.css"],
      ssr: "resources/js/ssr.js",
      {{- end}}
      publicDirectory: "public",
      buildDirectory: "build",
      refresh: true,
    }),
    {{- if .FrontendHasReact}}
    react({}),
    {{- end}}
    {{- if .FrontendHasVue}}
    vue({
      include: [/\.vue$/],
    }),
    {{- end}}
  ],
  {{- if .FrontendHasReact}}
  optimizeDeps: {
    force: true,
    esbuildOptions: {
      loader: {
        ".js": "jsx",
        ".ts": "tsx",
      },
    },
  },
  {{- end}}
  server: {
    hmr: {
      host: "localhost",
    },
  },
});
