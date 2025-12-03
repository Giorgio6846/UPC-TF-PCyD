import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // Load env vars from the project root (../.env) so Vite picks up the API keys placed there.
  envDir: '..'
})
