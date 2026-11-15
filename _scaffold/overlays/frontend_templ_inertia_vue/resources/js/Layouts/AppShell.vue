<script setup lang="ts">
import { Link, usePage } from "@inertiajs/vue3";

import Toasts from "../Components/Toasts.vue";

// The application shell: whatever is on every page.
//
// A skeleton on purpose — a header, a nav, a main and the toast stack. It is
// here to be rewritten, and what matters is where it sits rather than what it
// looks like: Layouts/resolve.ts makes this a *persistent* layout, so it is
// mounted once and survives every client-side navigation.
//
// That is what makes this the right home for anything that should outlive a
// page: an open realtime connection, a cached unread count, a sidebar's scroll
// position. A layout applied inside each page is remounted on every visit and
// loses all three.
//
// Nothing here reads a browser global during setup. A component's setup runs
// in Node during server rendering, where window and document do not exist, and
// the failure is a 500 from the renderer — or worse, a silent fallback to an
// empty container that answers 200. Read globals in onMounted.

const page = usePage();
</script>

<template>
  <div class="min-h-full bg-slate-50 text-slate-900">
    <header class="border-b border-slate-200 bg-white">
      <div class="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
        <!--
          Link rather than <a>, which is the difference between a client-side
          visit and a full page load. An <a> here would throw away the
          persistent layout this component exists to be.
        -->
        <Link href="/" class="font-semibold">Home</Link>

        <nav aria-label="Main" class="flex items-center gap-4 text-sm">
          <!--
            aria-current, so a screen reader announces which page this is.
            Compared against the component name Inertia already resolved
            rather than against window.location, which does not exist on the
            server.
          -->
          <Link
            href="/"
            :aria-current="page.component === 'IndexVue' ? 'page' : undefined"
            class="text-slate-600 hover:text-slate-900"
          >
            Example
          </Link>
        </nav>
      </div>
    </header>

    <!--
      id="main" and a skip link belong here once there is more than one nav
      item worth skipping.
    -->
    <main class="mx-auto max-w-5xl px-4 py-8">
      <slot />
    </main>

    <!--
      Rendered here, so every page has the stack without asking. What *fills*
      it is wired in app.js instead — see lib/flash.ts for why a server flash
      cannot be read from a component. A page that opts out of this layout
      therefore has nowhere to show one, which is usually right for the
      sign-in form that opts out.
    -->
    <Toasts />
  </div>
</template>
