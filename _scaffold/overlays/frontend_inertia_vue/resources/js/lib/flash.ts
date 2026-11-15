import { router, type FlashData } from "@inertiajs/core";

import { toast, type ToastTone } from "./toast";

// Flash messages from the server, raised as toasts.
//
// A handler that redirects after doing something writes one in Go:
//
//	ctx := inertia.SetFlashValue(c.RequestContext(), "success", "Saved")
//	c.SetRequest(c.Request().WithContext(ctx))
//	return inertia.Redirect(c, "/things")
//
// and it arrives on the next page. This turns the four conventional keys into
// toasts; anything else in the map is left for a page to render itself.
//
// **Through Inertia's own event, and not by reading the page.** The obvious
// implementation — usePage().flash in a layout — is wrong in a way that is
// hard to see: the flash is a real field of the page object and arrives in the
// document, but Inertia's core deliberately blanks it on the page it hands
// out, so that going back in history does not re-announce a message about
// something that already happened. Reading it there got an empty object in
// React and, by timing alone, the real one in Vue. This file shipped doing
// exactly that.
//
// Worth knowing which messages belong here. A toast is right for the result of
// something that moved the reader somewhere else — "Saved", "Invitation sent".
// A message about the page somebody is already on reads better rendered in the
// page, next to the thing it is about, where it answers the question the page
// asks rather than floating over it.

const tones: Partial<Record<keyof FlashData, ToastTone>> = {
  success: "good",
  warning: "warn",
  error: "bad",
  info: "neutral",
};

/**
 * listenForFlash raises a toast for every flash message the server sends, for
 * as long as the page is open.
 *
 * One listener covers both cases, and that is worth knowing because the first
 * page looks like it needs its own handling and does not. Inertia writes every
 * page into history before announcing it and history carries no flash — so
 * that going back does not repeat a message about something that already
 * happened — but the announcement reads the page it was handed rather than the
 * one that came back out, so the message survives. Raising
 * `props.initialPage.flash` in the entry's setup as well, which is the obvious
 * belt and braces, shows every such message twice.
 *
 * router.on is a no-op where there is no window, so this is safe to call from
 * a module a server renderer also loads. The returned unsubscriber is dropped
 * on purpose: this listener lives as long as the tab.
 */
export function listenForFlash(): void {
  router.on("flash", (event) => {
    const flash = event.detail.flash;
    for (const [name, tone] of Object.entries(tones) as [keyof FlashData, ToastTone][]) {
      const message = flash[name];
      if (typeof message === "string" && message !== "") {
        toast(message, tone);
      }
    }
  });
}
