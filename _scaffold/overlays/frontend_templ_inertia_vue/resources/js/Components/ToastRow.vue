<script setup lang="ts">
import { Link } from "@inertiajs/vue3";
import { computed, onMounted, onUnmounted } from "vue";

import { dismiss, TOAST_MS, type Toast, type ToastTone } from "../lib/toast";

// One toast.
//
// Its own component rather than a branch inside Toasts.vue, because the timer
// belongs to the toast: a component per row gets mounted and unmounted with
// the thing it is counting down, so dismissing by click cannot leave a timer
// behind that later dismisses whatever took its place. A Vue single-file
// component holds one component, which is the only reason this is a second
// file where the React overlay has a second function.

const props = defineProps<{ toast: Toast }>();

const tones: Record<ToastTone, { className: string; glyph: string; label: string }> = {
  neutral: { className: "bg-slate-900 text-white", glyph: "i", label: "Note" },
  good: { className: "bg-emerald-600 text-white", glyph: "✓", label: "Success" },
  warn: { className: "bg-amber-500 text-black", glyph: "!", label: "Warning" },
  bad: { className: "bg-red-600 text-white", glyph: "✕", label: "Error" },
};

const tone = computed(() => tones[props.toast.tone] ?? tones.neutral);
const shell = computed(
  () => `pointer-events-auto w-full max-w-sm rounded-lg px-4 py-3 text-sm shadow-lg ${tone.value.className}`,
);

let timer: ReturnType<typeof setTimeout> | undefined;

onMounted(() => {
  timer = setTimeout(() => dismiss(props.toast.id), TOAST_MS);
});

onUnmounted(() => {
  clearTimeout(timer);
});

function close(event: Event) {
  // preventDefault as well, since a toast with an href is a link and this
  // button is inside it.
  event.preventDefault();
  event.stopPropagation();
  dismiss(props.toast.id);
}
</script>

<template>
  <!--
    A link when it has somewhere to go and a plain div when it does not. One
    element with :is rather than two branches, so the body below is written
    once — a copy of it would be the thing that drifts.
  -->
  <component
    :is="toast.href ? Link : 'div'"
    :href="toast.href"
    :class="shell"
    @click="toast.href ? dismiss(toast.id) : undefined"
  >
    <div class="flex items-start gap-3">
      <!--
        The glyph is paired with the tone rather than passed in, so a failure
        cannot arrive wearing a tick — and so the tone is carried by a shape
        as well as by a colour, which is what this needs to survive a
        colour-blind reader.
      -->
      <span aria-hidden="true" class="mt-px font-semibold">{{ tone.glyph }}</span>
      <span class="sr-only">{{ tone.label }}:</span>
      <span class="min-w-0 flex-1">{{ toast.message }}</span>
      <!--
        type="button", because this can sit inside a form on some pages and
        would otherwise submit it.
      -->
      <button
        type="button"
        aria-label="Dismiss"
        class="-mr-1 shrink-0 rounded px-1 opacity-70 transition hover:opacity-100"
        @click="close"
      >
        ✕
      </button>
    </div>
  </component>
</template>
