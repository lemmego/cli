import { useEffect } from "react";

// useDocumentTitle sets the tab title on the client, and emits nothing on the
// server.
//
// **Use this rather than Inertia's `<Head title>`**, and the reason is a bug
// rather than a preference. With lemmego/seo the real <title> is rendered in
// Go — deliberately, so it survives a server-rendering failure, which answers
// 200 and looks perfect to a human while a crawler receives an empty
// container. `<Head title>` renders a second <title> into the same document on
// every successful render: a browser uses the first and a crawler reports a
// duplicate, for a tag whose only job is updating the title during a
// client-side visit.
//
// An effect runs only in the browser, which is exactly where that job is. On a
// client-side visit the root template is not re-rendered, so without this the
// tab keeps the title of the first page somebody landed on.
export function useDocumentTitle(title: string, suffix?: string) {
  useEffect(() => {
    if (!title) return;
    document.title = suffix ? `${title} · ${suffix}` : title;
  }, [title, suffix]);
}
