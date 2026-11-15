import { createSSRApp, h } from "vue";
import { renderToString } from "@vue/server-renderer";
import { createInertiaApp } from "@inertiajs/vue3";
import createServer from "@inertiajs/vue3/server";
import { resolvePageComponent } from "laravel-vite-plugin/inertia-helpers";

import { withShell } from "./Layouts/resolve";

// createServer is @inertiajs/core's own server — /render, /health, /shutdown and
// cluster mode — reached through @inertiajs/vue3/server. @inertiajs/vite's
// ssrTransform recognises this exact shape and rewrites it for the SSR bundle,
// so the form matters: do not restructure it.

createServer((page) =>
  createInertiaApp({
    page,
    render: renderToString,
    // The same layout the browser applies. A shell applied on one side only
    // is the hydration mismatch that reads as a page flashing on load.
    resolve: (name) =>
      resolvePageComponent(`./Pages/${name}.vue`, import.meta.glob("./Pages/**/*.vue")).then((module) =>
        withShell(module, name),
      ),
    setup({ App, props, plugin }) {
      return createSSRApp({ render: () => h(App, props) }).use(plugin);
    },
  }),
);
