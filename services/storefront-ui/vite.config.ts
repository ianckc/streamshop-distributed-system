import { defineConfig } from 'vite'
import { devtools } from '@tanstack/devtools-vite'

import { tanstackStart } from '@tanstack/react-start/plugin/vite'

import viteReact from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Public storefront at http://localhost:8080/ (Traefik PathPrefix `/`)
const base = process.env.STOREFRONT_UI_BASE || '/'

const config = defineConfig({
  base,
  resolve: { tsconfigPaths: true },
  server: {
    port: 3030,
    proxy: {
      // Same-origin /v1/* in the browser → StreamShop Traefik (or local gateway)
      '/v1': {
        target: process.env.AGENT_GATEWAY_PROXY || 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
  plugins: [devtools(), tailwindcss(), tanstackStart(), viteReact()],
})

export default config
