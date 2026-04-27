import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig(({ command, mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const normalizeBasePath = (value: string) => {
    const withDefault = value || '/'
    const withLeadingSlash = withDefault.startsWith('/') ? withDefault : `/${withDefault}`
    return withLeadingSlash.endsWith('/') ? withLeadingSlash : `${withLeadingSlash}/`
  }

  return {
    plugins: [react()],
    base: command === 'build' ? normalizeBasePath(env.VITE_BASE_PATH || '/pintour-travel/') : '/',
    server: {
      port: 3000,
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },
    build: {
      outDir: 'dist',
      sourcemap: false,
    },
  }
})
