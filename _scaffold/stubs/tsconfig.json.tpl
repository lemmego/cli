{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    {{- if .FrontendHasReact}}
    "jsx": "react-jsx",
    {{- end}}
    {{- if .FrontendHasVue}}
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
  "include": ["resources/js/**/*"]
}
