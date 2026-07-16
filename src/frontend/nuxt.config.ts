import tailwindcss from '@tailwindcss/vite'

export default defineNuxtConfig({
  devtools: { enabled: true },

  modules: ['@nuxt/fonts'],

  fonts: {
    families: [
      { name: 'IBM Plex Serif', provider: 'google', weights: [500, 600] },
      { name: 'IBM Plex Sans', provider: 'google', weights: [400, 500, 600] },
      { name: 'IBM Plex Mono', provider: 'google', weights: [400, 500] }
    ]
  },

  runtimeConfig: {
    // Server-only: used for requests made during SSR, which run inside the
    // Docker network and need the internal service hostname.
    apiBaseServer: process.env.NUXT_API_BASE_SERVER || 'http://api:8091/api/v1',
    public: {
      // Used for requests made from the browser (client-side navigation,
      // the create/delete forms) - needs a URL the browser can actually
      // reach, not the Docker-internal one above.
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8091/api/v1'
    }
  },

  app: {
    head: {
      title: 'Snoopy — saved searches',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Tracked listings from Czech second-hand marketplaces and eBay.' }
      ],
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }
      ]
    }
  },

  css: ['~/assets/css/main.css'],

  vite: {
    plugins: [tailwindcss()]
  },

  compatibilityDate: '2026-02-03'
})
