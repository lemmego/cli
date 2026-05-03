{
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    {{- if .FrontendHasReact}}
    "build:ssr": "vite build --ssr && node bootstrap/ssr/ssr.js",
    {{- end}}
    {{- if .FrontendHasVue}}
    "build:ssr": "vite build --ssr && node bootstrap/ssr/ssr.js",
    {{- end}}
    "preview": "vite preview"
  },
  "devDependencies": {
    "@tailwindcss/vite": "{{.Version "@tailwindcss/vite"}}",
    {{- if .FrontendHasReact}}
    "@types/react": "{{.Version "@types/react"}}",
    "@types/react-dom": "{{.Version "@types/react-dom"}}",
    "@vitejs/plugin-react": "{{.Version "@vitejs/plugin-react"}}",
    {{- end}}
    {{- if .FrontendHasVue}}
    "@vitejs/plugin-vue": "{{.Version "@vitejs/plugin-vue"}}",
    "@vue/server-renderer": "{{.Version "@vue/server-renderer"}}",
    {{- end}}
    "laravel-vite-plugin": "{{.Version "laravel-vite-plugin"}}",
    "tailwindcss": "{{.Version "tailwindcss"}}",
    "typescript": "{{.Version "typescript"}}",
    "vite": "{{.Version "vite"}}"
  },
  "dependencies": {
    {{- if .FrontendHasReact}}
    "@inertiajs/react": "{{.Version "@inertiajs/react"}}",
    "axios": "{{.Version "axios"}}",
    "react": "{{.Version "react"}}",
    "react-dom": "{{.Version "react-dom"}}"
    {{- end}}
    {{- if .FrontendHasVue}}
    "@inertiajs/vue3": "{{.Version "@inertiajs/vue3"}}",
    "axios": "{{.Version "axios"}}",
    "vue": "{{.Version "vue"}}"
    {{- end}}
  }
}
