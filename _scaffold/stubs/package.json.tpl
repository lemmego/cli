{
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    {{- if .InertiaProvider}}
    "build:ssr": "vite build --ssr && node bootstrap/ssr/ssr.js",
    {{- end}}
    "build:static": "npx @tailwindcss/cli -i static/css/style.css -o static/css/dist.css --minify",
    "dev:static": "npx @tailwindcss/cli -i static/css/style.css -o static/css/dist.css --minify --watch",
    "preview": "vite preview"
  },
  "devDependencies": {
    "@tailwindcss/cli": "{{.Version "@tailwindcss/cli"}}",
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
    {{- end}}
    {{- if .FrontendHasVue}}
    "@inertiajs/vue3": "{{.Version "@inertiajs/vue3"}}",
    {{- end}}
    {{- if .InertiaProvider}}
    "@inertiajs/vite": "{{.Version "@inertiajs/vite"}}",
    {{- end}}
    "axios": "{{.Version "axios"}}",
    {{- if .FrontendHasReact}}
    "react": "{{.Version "react"}}",
    "react-dom": "{{.Version "react-dom"}}"
    {{- end}}
    {{- if .FrontendHasVue}}
    "vue": "{{.Version "vue"}}"
    {{- end}}
  }
}
