import { useDocumentTitle } from "../Components/useDocumentTitle";
import { toast } from "../lib/toast";

// The example page.
//
// It renders inside AppShell without saying so: Layouts/resolve.tsx gives
// every page the shell unless it exports `layout = null`. A sign-in form is
// the usual reason to opt out.
//
// There is deliberately no `import React`. With react-jsx — which this
// project's tsconfig sets — JSX needs no import, and an unused one is a
// typecheck error rather than a harmless line.
const IndexReact = () => {
  // The tab title, set on the client only. With lemmego/seo the real <title>
  // comes from Go, so Inertia's <Head title> would send a second one on every
  // render. See Components/useDocumentTitle.ts.
  useDocumentTitle("Home", "Lemmego");

  return (
    <div className="mx-auto max-w-xl rounded-lg bg-white px-6 py-8 shadow ring-1 ring-slate-900/5">
      <div className="flex items-center justify-center">
        <img
          src="https://avatars.githubusercontent.com/u/109903896?s=200&v=4"
          alt="Lemmego"
          className="h-20 w-20"
        />
      </div>

      <div className="py-6 text-base leading-7">
        <p>
          <strong>Lemmego</strong> is a modern, full-stack web development
          framework built with Go.
        </p>

        <hr className="my-5 border-slate-200" />

        <p>
          <a
            href="https://lemmego.github.io"
            target="_blank"
            rel="noopener noreferrer"
            className="font-semibold text-red-500 hover:text-red-600"
          >
            Read the docs
          </a>{" "}
          to learn more about Lemmego.
        </p>
      </div>

      {/*
        The toast queue, raised from anywhere and rendered once by AppShell.
        A handler that redirects with a flash message needs none of this — see
        lib/flash.ts, which app.tsx already wires up.
      */}
      <button
        type="button"
        onClick={() => toast("That worked", "good")}
        className="rounded bg-slate-900 px-3 py-2 text-sm text-white hover:bg-slate-700"
      >
        Raise a toast
      </button>
    </div>
  );
};

export default IndexReact;
