# Vendored from `lemmego/broadcast`

`broadcast.ts` is a byte-identical copy of `broadcast/client/broadcast.ts`, and
`useBroadcast.ts` of whichever of `broadcast/client/useBroadcast.ts` and
`useBroadcastVue.ts` this project's frontend can compile.

They are vendored rather than installed because **the framework publishes no
npm package**. Go modules carry Go code; there is no distribution path for a
`.ts` file, so the scaffold copies it in. That is also why editing these here
is fine for your project and pointless as a contribution: the upstream copy is
the one other projects get.

The split is a dependency, not a preference: `broadcast.ts` imports nothing, so
it is safe in any project, while the hook imports React and the composable
imports Vue — and `tsconfig.json` typechecks everything under `resources/js`,
so the wrong one is a typecheck error in a file nobody touched. Both export one
`useBroadcast` and are written to the same path, so a framework is a different
file rather than a different call, and a project with no frontend at all gets
the client without either.

## The one rule worth keeping if you rewrite them

Nothing touches `window`, `document`, `EventSource` or `WebSocket` during a
render, and nothing at module scope touches them at all. A server-side render
runs this code in Node, where none of those exist — and the failure is a 500
from the renderer, or worse a silent fallback to an empty container that
answers 200. Read browser globals inside an effect, or in `onMounted`.
