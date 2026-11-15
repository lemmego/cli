import { createApp, h } from "vue";
import { createInertiaApp } from "@inertiajs/vue3";
import { resolvePageComponent } from "laravel-vite-plugin/inertia-helpers";

import { listenForFlash } from "./lib/flash";
import { withShell } from "./Layouts/resolve";
import "../css/app.css";

// Every flash message the server sends, for as long as this tab is open —
// including the one that arrived with this document. See lib/flash.ts, and in
// particular why there is nothing to do in setup for the first page.
listenForFlash();

createInertiaApp({
  resolve: (name) =>
    // Where a page gets its layout. See Layouts/resolve.ts — doing it here
    // rather than in each page is what makes the shell inherited rather than
    // remembered, and what makes it persistent across a client-side visit.
    //
    // In a `then`, because resolvePageComponent is asynchronous: this glob is
    // lazy, so a page is fetched when somebody first visits it instead of
    // being bundled into the entry. The React overlay globs eagerly and so
    // applies the shell inline.
    resolvePageComponent(`./Pages/${name}.vue`, import.meta.glob("./Pages/**/*.vue")).then((module) =>
      withShell(module, name),
    ),
  setup({ el, App, props, plugin }) {
    return createApp({ render: () => h(App, props) })
      .use(plugin)
      .mount(el);
  },
});
