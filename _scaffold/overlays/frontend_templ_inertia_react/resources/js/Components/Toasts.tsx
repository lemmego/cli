import { Link } from "@inertiajs/react";
import { useEffect, useState } from "react";

import { dismiss, subscribe, TOAST_MS, type Toast, type ToastTone } from "../lib/toast";

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

const tones: Record<ToastTone, { className: string; glyph: string; label: string }> = {
  neutral: { className: "bg-slate-900 text-white", glyph: "i", label: "Note" },
  good: { className: "bg-emerald-600 text-white", glyph: "✓", label: "Success" },
  warn: { className: "bg-amber-500 text-black", glyph: "!", label: "Warning" },
  bad: { className: "bg-red-600 text-white", glyph: "✕", label: "Error" },
};

export function Toasts() {
  const [toasts, setToasts] = useState<Toast[]>([]);

  // Subscribed in an effect rather than read during render. The store is plain
  // memory, but a render that read it would differ between the server and the
  // client the moment anything had been raised.
  useEffect(() => subscribe(setToasts), []);

  if (toasts.length === 0) return null;

  return (
    /*
      aria-live="polite" rather than "assertive": a toast is worth announcing
      and is not worth interrupting whatever a screen reader is already
      saying. The container stays in the tree while anything is in it, so an
      addition is announced on its own instead of the whole region being
      re-read.

      pointer-events-none on the container and auto on each toast, so the
      empty space around the stack stays clickable.
    */
    <div
      aria-live="polite"
      aria-atomic="false"
      className="pointer-events-none fixed inset-x-0 bottom-0 z-50 flex flex-col items-center gap-2 px-4 pb-4 sm:items-end sm:pr-6"
    >
      {toasts.map((one) => (
        <Row key={one.id} toast={one} />
      ))}
    </div>
  );
}

function Row({ toast }: { toast: Toast }) {
  // One timer per toast, cleared on unmount — so dismissing by click does not
  // leave a timer behind that later dismisses whatever took its place.
  useEffect(() => {
    const timer = setTimeout(() => dismiss(toast.id), TOAST_MS);
    return () => clearTimeout(timer);
  }, [toast.id]);

  const tone = tones[toast.tone] ?? tones.neutral;

  const body = (
    <div className="flex items-start gap-3">
      {/*
        The glyph is paired with the tone rather than passed in, so a failure
        cannot arrive wearing a tick — and so the tone is carried by a shape as
        well as by a colour, which is what this needs to survive a
        colour-blind reader.
      */}
      <span aria-hidden="true" className="mt-px font-semibold">
        {tone.glyph}
      </span>
      <span className="sr-only">{tone.label}:</span>
      <span className="min-w-0 flex-1">{toast.message}</span>
      {/*
        type="button", because this can sit inside a form on some pages and
        would otherwise submit it. preventDefault as well, since a toast with
        an href is a link and the button is inside it.
      */}
      <button
        type="button"
        aria-label="Dismiss"
        onClick={(event) => {
          event.preventDefault();
          event.stopPropagation();
          dismiss(toast.id);
        }}
        className="-mr-1 shrink-0 rounded px-1 opacity-70 transition hover:opacity-100"
      >
        ✕
      </button>
    </div>
  );

  const shell = `pointer-events-auto w-full max-w-sm rounded-lg px-4 py-3 text-sm shadow-lg ${tone.className}`;

  if (toast.href) {
    return (
      <Link href={toast.href} className={shell} onClick={() => dismiss(toast.id)}>
        {body}
      </Link>
    );
  }
  return <div className={shell}>{body}</div>;
}
