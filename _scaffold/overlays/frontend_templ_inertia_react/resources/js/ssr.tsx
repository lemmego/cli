import type { Page } from "@inertiajs/core";
import { createInertiaApp } from "@inertiajs/react";
import createSSRServer from "@inertiajs/react/server";
import ReactDOMServer from "react-dom/server";
import { resolvePageComponent } from "@lemmego/vite/inertia-helpers";
import { pages as packagePages } from "virtual:lemmego/pages";

import { withShell } from "./Layouts/resolve";
import "../css/app.css";

// Keep the server import aliased: @inertiajs/vite rewrites a top-level call
// named createServer and drops its options. This runtime option keeps the
// listener aligned with the URL configured by `lemmego dev`.
const renderPage = (page: Page) =>
  createInertiaApp({
    page,
    render: ReactDOMServer.renderToString,
    resolve: (name) => {
      const pages = {
        ...packagePages,
        ...Object.fromEntries(Object.entries(import.meta.glob("./Pages/**/*.tsx")).map(([path, loader]) => [
          path.replace(/^\.\/Pages\//, "").replace(/\.tsx$/, ""),
          loader,
        ])),
      };
      if (!pages[name]) throw new Error(`Inertia SSR: no page component at ${name}`);
      return resolvePageComponent(name, pages).then((module) => withShell(module, name) as never);
    },
    setup: ({ App, props }) => <App {...props} />,
  });

const runtimeEnv = (globalThis as typeof globalThis & { process?: { env?: Record<string, string | undefined> } }).process?.env ?? {};
const ssrPort = Number(runtimeEnv.INERTIA_SSR_PORT || 13714);
const ssrDisabled = ["false", "0", "f"].includes((runtimeEnv.INERTIA_SSR || "").trim().toLowerCase());
if (import.meta.env.PROD && !ssrDisabled) {
  createSSRServer(renderPage, { port: ssrPort });
}

export default renderPage;
