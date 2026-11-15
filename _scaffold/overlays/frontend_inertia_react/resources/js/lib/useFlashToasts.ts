import { usePage } from "@inertiajs/react";
import { useEffect, useRef } from "react";

import { toast, type ToastTone } from "./toast";

// Flash messages from the server, raised as toasts.
//
// A handler that redirects after doing something writes one in Go:
//
//	ctx := inertia.SetFlashValue(c.RequestContext(), "success", "Saved")
//	c.SetRequest(c.Request().WithContext(ctx))
//	return inertia.Redirect(c, "/things")
//
// and it arrives on the next page as page.flash. This turns the four
// conventional keys into toasts; anything else in the map is left for a page
// to render itself.
//
// Worth knowing which messages belong here. A toast is right for the result of
// something that moved the reader somewhere else — "Saved", "Invitation sent".
// A message about the page somebody is already on reads better rendered in the
// page, next to the thing it is about, where it answers the question the page
// asks rather than floating over it.

type Flash = Partial<Record<string, unknown>>;

const tones: Record<string, ToastTone> = {
  success: "good",
  warning: "warn",
  error: "bad",
  info: "neutral",
};

export function useFlashToasts() {
  // usePage re-renders on every visit, and the flash for one visit must be
  // raised once. Keyed on the page's own id rather than on the message, so two
  // identical messages from two visits are two toasts — which is what somebody
  // who clicked Save twice should see.
  const page = usePage();
  const flash = (page.props.flash ?? {}) as Flash;
  const seen = useRef<string | null>(null);

  useEffect(() => {
    const key = page.component + "|" + JSON.stringify(flash);
    if (seen.current === key) return;
    seen.current = key;

    for (const [name, tone] of Object.entries(tones)) {
      const message = flash[name];
      if (typeof message === "string" && message !== "") {
        toast(message, tone);
      }
    }
  });
}
