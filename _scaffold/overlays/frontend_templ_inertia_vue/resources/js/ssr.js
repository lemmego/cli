import { createSSRApp, h } from "vue";
import { renderToString } from "@vue/server-renderer";
import { createInertiaApp } from "@inertiajs/vue3";
import createSSRServer from "@inertiajs/vue3/server";
import { resolvePageComponent } from "@lemmego/vite/inertia-helpers";
import { pages as packagePages } from "virtual:lemmego/pages";

import { withShell } from "./Layouts/resolve";

// Keep the server import aliased: @inertiajs/vite rewrites a top-level call
// named createServer and drops its options. This runtime option keeps the
// listener aligned with the URL configured by `lemmego dev`.
const renderPage = (page) =>
  createInertiaApp({
    page,
    render: renderToString,
    // The same layout the browser applies. A shell applied on one side only
    // is the hydration mismatch that reads as a page flashing on load.
    resolve: (name) =>
      resolvePageComponent(name, {
        ...packagePages,
        ...Object.fromEntries(Object.entries(import.meta.glob("./Pages/**/*.vue")).map(([path, loader]) => [
          path.replace(/^\.\/Pages\//, "").replace(/\.vue$/, ""),
          loader,
        ])),
      }).then((module) =>
        withShell(module, name),
      ),
    setup({ App, props, plugin }) {
      return createSSRApp({ render: () => h(App, props) }).use(plugin);
    },
  });

const runtimeEnv = globalThis.process?.env ?? {};
const ssrPort = Number(runtimeEnv.INERTIA_SSR_PORT || 13714);
const ssrDisabled = ["false", "0", "f"].includes((runtimeEnv.INERTIA_SSR || "").trim().toLowerCase());
if (import.meta.env.PROD && !ssrDisabled) {
  createSSRServer(renderPage, { port: ssrPort });
}

export default renderPage;
