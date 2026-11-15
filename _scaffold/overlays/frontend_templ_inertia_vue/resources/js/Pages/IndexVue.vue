<script setup lang="ts">
import { useDocumentTitle } from "../Components/useDocumentTitle";
import { toast } from "../lib/toast";

// The example page.
//
// It renders inside AppShell without saying so: Layouts/resolve.ts gives every
// page the shell unless it declares a layout of its own. A sign-in form is the
// usual reason to opt out, which is a plain `<script>` block beside this one
// exporting `{ layout: noShell }` — see Layouts/resolve.ts, and in particular
// why the opt-out is a named constant rather than `null`.

// The tab title, set on the client only. With lemmego/seo the real <title>
// comes from Go, so Inertia's <Head title> would send a second one on every
// render. See Components/useDocumentTitle.ts.
useDocumentTitle("Home", "Lemmego");
</script>

<template>
  <div class="mx-auto max-w-xl rounded-lg bg-white px-6 py-8 shadow ring-1 ring-slate-900/5">
    <div class="flex items-center justify-center">
      <img
        src="https://avatars.githubusercontent.com/u/109903896?s=200&v=4"
        alt="Lemmego"
        class="h-20 w-20"
      />
    </div>

    <div class="py-6 text-base leading-7">
      <p><strong>Lemmego</strong> is a modern, full-stack web development framework built with Go.</p>

      <hr class="my-5 border-slate-200" />

      <p>
        <a
          href="https://lemmego.github.io"
          target="_blank"
          rel="noopener noreferrer"
          class="font-semibold text-red-500 hover:text-red-600"
        >
          Read the docs
        </a>
        to learn more about Lemmego.
      </p>
    </div>

    <!--
      The toast queue, raised from anywhere and rendered once by AppShell. A
      handler that redirects with a flash message needs none of this — see
      lib/flash.ts, which app.js already wires up.
    -->
    <button
      type="button"
      class="rounded bg-slate-900 px-3 py-2 text-sm text-white hover:bg-slate-700"
      @click="toast('That worked', 'good')"
    >
      Raise a toast
    </button>
  </div>
</template>
