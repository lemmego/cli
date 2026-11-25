import { defineConfig, loadEnv } from "vite";
import lemmego from "@lemmego/vite";
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

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  // Generated SSR entries read this at runtime from the same environment that
  // `lemmego dev` uses to choose the renderer URL.
  const ssrPort = Number(process.env.INERTIA_SSR_PORT || env.INERTIA_SSR_PORT || 13714);
  const ssrOverride = (process.env.INERTIA_SSR || env.INERTIA_SSR || "").trim().toLowerCase();
  const ssrDisabled = ["false", "0", "f"].includes(ssrOverride);
  if (!Number.isInteger(ssrPort) || ssrPort < 1 || ssrPort > 65535) {
    throw new Error("INERTIA_SSR_PORT must be a port from 1 to 65535");
  }
  const inertiaOptions = process.env.INERTIA_SSR_URL || ssrDisabled ? { ssr: false } : {};

  return {
  resolve: {
    alias: {
      "@": path.resolve("./resources/js"),
    },
  },
  plugins: [
    tailwindcss(),
    lemmego({
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
    inertia(inertiaOptions),
    {{- end}}
  ],
  {{- if .FrontendHasReact}}
  // Dependency prebundling is Rolldown's job as of Vite 8, not esbuild's, so
  // this is `rolldownOptions.moduleTypes` rather than `esbuildOptions.loader`.
  // The old key is not an error — Vite warns on every build and ignores it,
  // which is the shape of a setting that quietly stops applying.
  optimizeDeps: {
    rolldownOptions: {
      moduleTypes: {
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
  };
});
