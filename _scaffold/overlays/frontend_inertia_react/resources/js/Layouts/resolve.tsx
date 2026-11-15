import type { ComponentType, ReactNode } from "react";

import { AppShell } from "./AppShell";

// Resolving a page is also where it gets its layout, and it is the only place
// that should be.
//
// Inertia's persistent-layout convention is a `layout` property on the page
// component. Setting it here rather than in each page is what makes the shell
// inherited rather than remembered: thirty pages each importing and wrapping
// AppShell is thirty chances to forget, and a page that forgets renders with
// no navigation at all and nothing to say so.
//
// Persistent also means the shell is not unmounted between visits, which is
// what keeps a scroll position, an open menu and a realtime connection alive
// across a client-side navigation. A layout applied inside each page is
// remounted on every visit and loses all three.

type PageModule = {
  default: ComponentType<Record<string, unknown>> & {
    layout?: ((page: ReactNode) => ReactNode) | null;
  };
};

/**
 * withShell gives a page the application shell unless it opts out.
 *
 * A page opts out with `export const layout = null`, which is what a sign-in
 * form wants: the chrome offers a signed-out visitor a navigation rail they
 * cannot use.
 */
export function withShell(module: unknown, name: string): unknown {
  const page = (module as PageModule).default;
  if (!page) return module;

  // `undefined` means "nothing said", which inherits. `null` means
  // "deliberately none", which does not — so this tests for the property's
  // presence and never for its truthiness. `if (!page.layout)` would be the
  // same expression for both and would quietly put the shell back around
  // every page that asked not to have it.
  if (!("layout" in page)) {
    page.layout = (content: ReactNode) => <AppShell>{content}</AppShell>;
  }

  // Unused today, and taken so that a rule keyed on the page's path — a
  // different shell under Admin/, say — has somewhere to live rather than
  // being bolted onto the call site later.
  void name;

  return module;
}
