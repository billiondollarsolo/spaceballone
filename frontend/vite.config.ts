import { defineConfig } from 'vite'
import { tanstackStart } from '@tanstack/react-start/plugin/vite'
import viteTsConfigPaths from 'vite-tsconfig-paths'
import tailwindcss from '@tailwindcss/vite'
import viteReact from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [
    viteTsConfigPaths(),
    tanstackStart({
      srcDirectory: 'app',
    }),
    viteReact(),
    tailwindcss(),
  ],
})
