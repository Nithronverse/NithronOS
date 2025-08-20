import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'
import path from 'path'

export default defineConfig({
	plugins: [react()],
	resolve: { alias: { '@': path.resolve(__dirname, 'src'), '@/api-types': path.resolve(__dirname, 'src/api/schema.ts') } },
	server: {
		proxy: {
			'/api': {
				target: 'http://127.0.0.1:9000',
				changeOrigin: true,
			},
		},
	},
})


