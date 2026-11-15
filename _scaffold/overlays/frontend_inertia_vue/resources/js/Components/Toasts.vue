<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue";

import { subscribe, type Toast } from "../lib/toast";
import ToastRow from "./ToastRow.vue";

// The toast stack.
//
// Rendered once, in the layout, so every page has it without asking. On the
// server the queue is empty and this renders nothing — which is also the
// client's first render, so there is nothing for hydration to disagree about.
//
// Styled with plain Tailwind classes and an inline glyph rather than an icon
// package, so this file has no dependency beyond what a generated project
// already installs. Replace it with your own component library when you add
// one; the store in lib/toast.ts is what the rest of the application talks to.

const toasts = ref<Toast[]>([]);
let unsubscribe: (() => void) | null = null;

// Subscribed on mount rather than during setup. The store is plain memory and
// reading it in setup would compile — but setup runs in the server renderer,
// and a queue read there is whatever the process happened to be holding.
onMounted(() => {
  unsubscribe = subscribe((next) => {
    toasts.value = next;
  });
});

onUnmounted(() => {
  unsubscribe?.();
  unsubscribe = null;
});
</script>

<template>
  <!--
    aria-live="polite" rather than "assertive": a toast is worth announcing
    and is not worth interrupting whatever a screen reader is already saying.
    The container stays in the tree while anything is in it, so an addition is
    announced on its own instead of the whole region being re-read.

    pointer-events-none on the container and auto on each toast, so the empty
    space around the stack stays clickable.
  -->
  <div
    v-if="toasts.length > 0"
    aria-live="polite"
    aria-atomic="false"
    class="pointer-events-none fixed inset-x-0 bottom-0 z-50 flex flex-col items-center gap-2 px-4 pb-4 sm:items-end sm:pr-6"
  >
    <ToastRow v-for="one in toasts" :key="one.id" :toast="one" />
  </div>
</template>
