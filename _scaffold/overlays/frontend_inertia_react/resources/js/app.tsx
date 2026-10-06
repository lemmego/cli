import { createInertiaApp } from "@inertiajs/react";
import { createRoot } from "react-dom/client";
import "../css/app.css";

// The setup argument is deliberately not annotated.
//
// A hand-written {el, App, props} type does not match Inertia's own — props is
// InertiaAppProps, not Record<string, unknown>, and el is nullable in two of the
// three overloads — so annotating it makes `tsc --noEmit` fail on this file while
// Vite, which strips types with esbuild and never typechecks, builds it happily.
// Inference gets this exactly right.

createInertiaApp({
  resolve: (name) => {
    const pages = import.meta.glob("./Pages/**/*.tsx", { eager: true });
    const page = pages[`./Pages/${name}.tsx`];
    if (!page) {
      throw new Error(`Inertia: no page component at ./Pages/${name}.tsx`);
    }
    return page as never;
  },
  setup({ el, App, props }) {
    createRoot(el).render(<App {...props} />);
  },
});
