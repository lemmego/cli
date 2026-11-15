import { onMounted, toValue, watchEffect, type MaybeRefOrGetter } from "vue";

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
// A mounted hook runs only in the browser, which is exactly where that job is.
// On a client-side visit the root template is not re-rendered, so without this
// the tab keeps the title of the first page somebody landed on.
//
// The arguments are refs, getters or plain strings, so a title derived from a
// prop keeps the tab up to date without the caller watching it: the effect
// tracks whatever `toValue` read.
export function useDocumentTitle(
  title: MaybeRefOrGetter<string>,
  suffix?: MaybeRefOrGetter<string | undefined>,
) {
  onMounted(() => {
    watchEffect(() => {
      const name = toValue(title);
      if (!name) return;
      const tail = toValue(suffix);
      document.title = tail ? `${name} · ${tail}` : name;
    });
  });
}
