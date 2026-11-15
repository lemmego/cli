import type { Component } from "vue";

import AppShell from "./AppShell.vue";

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
//
// In Vue the layout is the component itself — @inertiajs/vue3 also takes an
// array of components to nest them, or a render function — and the page is
// rendered into its default slot. The React adapter instead takes a function
// of the page's own element, which is why AppShell.vue has a <slot /> where
// the React one takes children.

type PageModule = {
  default: Component & { layout?: unknown };
};

/**
 * noShell is how a page opts out, in a plain `<script>` block beside its
 * `<script setup>`:
 *
 *	export default { layout: noShell };
 *
 * which is what a sign-in form wants: the chrome offers a signed-out visitor a
 * navigation rail they cannot use.
 *
 * It is `undefined` rather than `null`, and that is a type constraint rather
 * than a preference: @inertiajs/vue3 augments Vue's component options with a
 * `layout` whose type does not admit `null`, so the React overlay's
 * `layout = null` is a `vue-tsc` error here. The *presence* of the property is
 * what withShell reads, so a declared `undefined` says "deliberately none"
 * where saying nothing at all means "inherit" — and a named constant is what
 * keeps that legible at the call site, since a bare `layout: undefined` reads
 * like a line that does nothing.
 */
export const noShell = undefined;

/**
 * withShell gives a page the application shell unless it opts out.
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
    page.layout = AppShell;
  }

  // Unused today, and taken so that a rule keyed on the page's path — a
  // different shell under Admin/, say — has somewhere to live rather than
  // being bolted onto the call site later.
  void name;

  return module;
}
