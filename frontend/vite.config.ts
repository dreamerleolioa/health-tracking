import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit(),
		SvelteKitPWA({
			strategies: 'generateSW',
			registerType: 'autoUpdate',
			manifest: {
				name: '健康追蹤計劃',
				short_name: 'Health Tracker',
				description: '全方位生活與健康追蹤儀表板',
				theme_color: '#E4000F',
				background_color: '#1a1a2e',
				display: 'standalone',
				icons: [
					{ src: '/favicon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any maskable' }
				]
			},
			workbox: {
				globPatterns: ['**/*.{js,css,html,svg,png,ico}'],
				runtimeCaching: [
					{
						urlPattern: ({ url }) => url.pathname.startsWith('/v1/'),
						handler: 'NetworkFirst',
						options: {
							cacheName: 'api-cache',
							networkTimeoutSeconds: 10,
							expiration: { maxEntries: 50, maxAgeSeconds: 5 * 60 }
						}
					}
				]
			}
		})
	]
});
