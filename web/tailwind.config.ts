import type { Config } from 'tailwindcss'

export default {
	darkMode: ['class'],
	content: ['./index.html', './src/**/*.{ts,tsx}'],
	theme: {
		extend: {
			colors: {
				background: '#111318',
				foreground: '#E7ECF3',
				card: '#191C21',
				primary: {
					DEFAULT: '#2D7FF9',
					foreground: '#0A0C10',
				},
				accent: {
					DEFAULT: '#A4F932',
					foreground: '#0A0C10',
				},
				muted: '#2A2E35',
				'muted-foreground': '#B9C3D1',
			},
			borderRadius: {
				lg: '0.5rem',
				md: '0.375rem',
				sm: '0.25rem',
			},
		},
	},
	plugins: [require('tailwindcss-animate')],
} satisfies Config


