export default defineNuxtConfig({
  devtools: { enabled: true },

  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://api:8091/api/v1'
    }
  },

  app: {
    head: {
      title: 'Second-Hand Shop Scraper',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Search and browse products from Czech second-hand marketplaces' }
      ],
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }
      ]
    }
  },

  css: ['~/assets/css/main.css'],

  modules: ['@nuxtjs/tailwindcss'],

  compatibilityDate: '2026-02-03'
})
