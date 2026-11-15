# Vendored from `lemmego/broadcast`

`broadcast.ts` is a byte-identical copy of `broadcast/client/broadcast.ts`,
and `useBroadcast.ts` — written only for a React project — of
`broadcast/client/useBroadcast.ts`.

They are vendored rather than installed because **the framework publishes no
npm package**. Go modules carry Go code; there is no distribution path for a
`.ts` file, so the scaffold copies it in. That is also why editing these here
is fine for your project and pointless as a contribution: the upstream copy is
the one other projects get.

The split is a dependency, not a preference: `broadcast.ts` imports nothing, so
it is safe in any project, while `useBroadcast.ts` imports React — which a Vue
project does not have, and `tsconfig.json` typechecks everything under
`resources/js`. So the hook is written only for a React frontend, and a Vue
project gets the client without it.

## The one rule worth keeping if you rewrite them

Nothing touches `window`, `document`, `EventSource` or `WebSocket` during a
render, and nothing at module scope touches them at all. A server-side render
runs this code in Node, where none of those exist — and the failure is a 500
from the renderer, or worse a silent fallback to an empty container that
answers 200. Read browser globals inside an effect.
