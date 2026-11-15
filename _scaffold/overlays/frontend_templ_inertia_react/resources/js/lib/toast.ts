// Transient messages, for the things that happen while somebody is looking at
// something else.
//
// Not a replacement for a message rendered beside the thing it is about. "Your
// address was changed" belongs on the email page, where it answers the question
// that page asks; a toast is for an event with no place on the page somebody is
// currently reading — a background job finishing, a realtime notification
// arriving.
//
// A module-level store rather than a context provider, because there is no tree
// to scope it to: a toast raised by a hook in the layout and one raised by a
// page are the same queue, and a provider would make them two.
//
// Nothing here touches a browser global, so the module is inert under
// renderToString. The server renders an empty list, which is also what the
// client renders first — so there is no hydration mismatch to reconcile.

export type ToastTone = "neutral" | "good" | "warn" | "bad";

export type Toast = {
  id: number;
  message: string;
  tone: ToastTone;
  /** Where clicking it goes, when that makes sense. */
  href?: string;
};

type Listener = (toasts: Toast[]) => void;

// Capped, because a burst — a reconnect storm, a batch of notifications —
// would otherwise stack a column taller than the window. The oldest go, since
// the newest are the ones nobody has read.
const MAX_VISIBLE = 4;

/** How long one stays: long enough to read a sentence, short enough to leave. */
export const TOAST_MS = 6000;

let nextID = 1;
let current: Toast[] = [];
const listeners = new Set<Listener>();

function publish() {
  // A fresh array each time, so a consumer comparing by reference re-renders.
  // Mutating and publishing the same array is the bug where the first toast
  // appears and the second never does.
  const snapshot = current.slice();
  for (const listener of listeners) listener(snapshot);
}

/** Raise a toast. Returns its id, so a caller can dismiss it early. */
export function toast(message: string, tone: ToastTone = "neutral", href?: string): number {
  const id = nextID++;
  current = [...current, { id, message, tone, href }];
  if (current.length > MAX_VISIBLE) {
    current = current.slice(current.length - MAX_VISIBLE);
  }
  publish();
  return id;
}

/** Dismiss one. An unknown id is ignored, because a timer and a click race. */
export function dismiss(id: number) {
  const next = current.filter((one) => one.id !== id);
  if (next.length === current.length) return;
  current = next;
  publish();
}

/** Subscribe to the queue. Returns an unsubscriber. */
export function subscribe(listener: Listener): () => void {
  listeners.add(listener);
  // Called immediately, so a component mounting after a toast was raised still
  // shows it — which is what happens on the first render after a visit.
  listener(current.slice());
  return () => {
    listeners.delete(listener);
  };
}

/** Clear everything. For a test, and for a sign-out. */
export function clear() {
  if (current.length === 0) return;
  current = [];
  publish();
}
