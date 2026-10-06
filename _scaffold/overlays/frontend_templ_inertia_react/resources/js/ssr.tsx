import { createInertiaApp } from "@inertiajs/react";
import createServer from "@inertiajs/react/server";
import ReactDOMServer from "react-dom/server";
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
      const pages = import.meta.glob("./Pages/**/*.tsx", { eager: true });
      const component = pages[`./Pages/${name}.tsx`];
      if (!component) {
        throw new Error(`Inertia SSR: no page component at ./Pages/${name}.tsx`);
      }
      return component as never;
    },
    setup: ({ App, props }) => <App {...props} />,
  }),
);
