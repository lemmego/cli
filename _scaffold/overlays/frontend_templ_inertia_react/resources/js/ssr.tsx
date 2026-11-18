import { createInertiaApp } from "@inertiajs/react";
import createServer from "@inertiajs/react/server";
import ReactDOMServer from "react-dom/server";
import { resolvePageComponent } from "@lemmego/vite/inertia-helpers";
import { pages as packagePages } from "virtual:lemmego/pages";

import { withShell } from "./Layouts/resolve";
import "../css/app.css";

// createServer is @inertiajs/core's own server — /render, /health, /shutdown and
// cluster mode — reached through @inertiajs/react/server. @inertiajs/vite's
// ssrTransform recognises this exact shape and rewrites it for the SSR bundle,
// so the form matters: do not restructure it.
//
// The setup argument is unannotated for the reason given in app.tsx.

createServer((page) =>
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
  }),
);
