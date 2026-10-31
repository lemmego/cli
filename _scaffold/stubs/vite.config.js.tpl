import { defineConfig } from "vite";
import laravel from "laravel-vite-plugin";
{{- if .FrontendHasReact}}
import react from "@vitejs/plugin-react";
{{- end}}
{{- if .FrontendHasVue}}
import vue from "@vitejs/plugin-vue";
{{- end}}
import tailwindcss from "@tailwindcss/vite";
{{- if .InertiaProvider}}
import inertia from "@inertiajs/vite";
{{- end}}
import path from "path";

export default defineConfig(({ command }) => ({
  resolve: {
    alias: {
      "@": path.resolve("./resources/js"),
    },
  },
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
    {{- if .InertiaProvider}}
    // `lemmego dev` sets INERTIA_SSR_URL when it runs the standalone renderer
    // from bootstrap/ssr/ssr.js. With something else rendering, this plugin's
    // own dev endpoint is dead weight that is not free: it pulls every page
    // component through ssrLoadModule the moment the dev server starts
    // listening, on the one event loop that is also serving the client bundle.
    // Measured on a real application, a first page load spent nine seconds
    // waiting for its stylesheet behind that warm-up; with the endpoint off,
    // the same request took 0.03s.
    //
    // `command === "build"` keeps it fully enabled for `vite build --ssr`,
    // which is what produces bootstrap/ssr/ssr.js: the plugin wraps the entry
    // with createServer and sets the rollup input, so disabling it there would
    // leave no standalone renderer to point at.
    inertia(command === "build" || !process.env.INERTIA_SSR_URL ? {} : { ssr: false }),
    {{- end}}
  ],
  {{- if .FrontendHasReact}}
  optimizeDeps: {
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
}));
