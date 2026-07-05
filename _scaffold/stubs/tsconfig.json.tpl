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
    "skipLibCheck": true
  },
  "include": ["resources/js/**/*"]
}
