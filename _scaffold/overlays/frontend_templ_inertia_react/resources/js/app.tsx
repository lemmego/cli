import { createInertiaApp } from "@inertiajs/react";
import { createRoot } from "react-dom/client";

import { listenForFlash } from "./lib/flash";
import { withShell } from "./Layouts/resolve";
import "../css/app.css";

// The setup argument is deliberately not annotated.
//
// A hand-written {el, App, props} type does not match Inertia's own — props is
// InertiaAppProps, not Record<string, unknown>, and el is nullable in two of the
// three overloads — so annotating it makes `tsc --noEmit` fail on this file while
// Vite, which strips types with esbuild and never typechecks, builds it happily.
// Inference gets this exactly right.

// Every flash message the server sends, for as long as this tab is open —
// including the one that arrived with this document. See lib/flash.ts, and in
// particular why there is nothing to do in setup for the first page.
listenForFlash();

createInertiaApp({
  resolve: (name) => {
    const pages = import.meta.glob("./Pages/**/*.tsx", { eager: true });
    const page = pages[`./Pages/${name}.tsx`];
    if (!page) {
      throw new Error(`Inertia: no page component at ./Pages/${name}.tsx`);
    }
    // Where a page gets its layout. See Layouts/resolve.tsx — doing it here
    // rather than in each page is what makes the shell inherited rather than
    // remembered, and what makes it persistent across a client-side visit.
    return withShell(page, name) as never;
  },
  setup({ el, App, props }) {
    createRoot(el).render(<App {...props} />);
  },
});
