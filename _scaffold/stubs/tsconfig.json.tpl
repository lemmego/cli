{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    {{- if .FrontendHasReact}}
    "jsx": "react-jsx",
    {{- end}}
    {{- if .FrontendHasVue}}
    // Vue compiles its own templates, so JSX is left alone rather than
    // transformed.
    "jsx": "preserve",
    {{- end}}
    "strict": true,
    "paths": {
      "@/*": ["./resources/js/*"]
    },
    "baseUrl": ".",
    "skipLibCheck": true,
    "noEmit": true,
    // Vite's ambient types declare import.meta.glob and import.meta.env.
    // Without them an entry point fails to typecheck on its very first line,
    // which is easy to miss because Vite strips types with esbuild and never
    // typechecks at all. This file is only written for a project that has Vite.
    "types": ["vite/client"]
  },
  {{- if .FrontendHasVue}}
  // The typecheck is package.json's `typecheck` script, which is `vue-tsc`.
  //
  // Plain `tsc` cannot do this job and does not say so usefully: it has no
  // idea what a .vue file is, so every import of one is reported as a missing
  // module and nothing inside a <script setup> block is read at all. Vite
  // never typechecks either — esbuild strips the types — so a type error in a
  // component builds and ships. That combination is how a scaffolded project
  // once failed its own typecheck on the first file for two releases.
  {{- end}}
  "include": ["resources/js/**/*"]
}
