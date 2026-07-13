<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Header -->
    <header class="bg-gradient-to-r from-purple-600 to-indigo-600 text-white shadow-lg">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <h1 class="text-4xl font-bold mb-2">🔍 Second-Hand Shop Scraper</h1>
        <p class="text-purple-100 text-lg">Browse products from Czech second-hand marketplaces</p>
      </div>
    </header>

    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Loading State -->
      <div v-if="pending" class="flex items-center justify-center py-12">
        <div class="text-center">
          <div class="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
          <p class="mt-4 text-gray-600 text-lg">Loading searches...</p>
        </div>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="bg-red-50 border-l-4 border-red-500 p-6 rounded-lg shadow-md">
        <div class="flex items-start">
          <div class="flex-shrink-0">
            <svg class="h-6 w-6 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <div class="ml-3">
            <h3 class="text-lg font-medium text-red-800">Error loading searches</h3>
            <p class="mt-2 text-red-700">{{ error.message }}</p>
            <p class="mt-2 text-sm text-red-600">Make sure the API server is running on http://localhost:8091</p>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else-if="!searches || searches.length === 0" class="bg-white rounded-xl shadow-lg p-12 text-center">
        <div class="mx-auto flex items-center justify-center h-20 w-20 rounded-full bg-purple-100 mb-6">
          <svg class="h-10 w-10 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
        <h2 class="text-2xl font-bold text-gray-900 mb-4">No searches yet</h2>
        <p class="text-gray-600 mb-4">Start by running a search from the command line:</p>
        <div class="bg-gray-50 rounded-lg p-4 mb-4">
          <code class="text-sm text-gray-800 font-mono">./search -keyword="hemingway"</code>
        </div>
        <p class="text-gray-600 mb-2">Or with Docker:</p>
        <div class="bg-gray-50 rounded-lg p-4">
          <code class="text-sm text-gray-800 font-mono">docker compose exec api ./search -keyword="hemingway"</code>
        </div>
      </div>

      <!-- Searches List -->
      <div v-else>
        <div class="mb-8">
          <h2 class="text-3xl font-bold text-gray-900">
            Found {{ searches.length }} search{{ searches.length !== 1 ? 'es' : '' }}
          </h2>
          <p class="mt-2 text-gray-600">Click on a search to view products</p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <NuxtLink
            v-for="search in searches"
            :key="search.id"
            :to="`/search/${search.id}`"
            class="group bg-white rounded-xl shadow-md hover:shadow-xl transition-all duration-300 overflow-hidden hover:-translate-y-1"
          >
            <div class="p-6">
              <div class="flex items-start justify-between mb-4">
                <h3 class="text-xl font-bold text-purple-600 group-hover:text-purple-700 break-words flex-1">
                  {{ search.keyword }}
                </h3>
                <svg class="h-5 w-5 text-purple-400 group-hover:text-purple-600 transition-colors flex-shrink-0 ml-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
              </div>

              <div class="flex items-center text-sm text-gray-500">
                <svg class="h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <span>{{ formatDate(search.created_at) }}</span>
              </div>

              <div class="mt-4">
                <span class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-gradient-to-r from-purple-600 to-indigo-600 text-white">
                  View Products →
                </span>
              </div>
            </div>
          </NuxtLink>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
const config = useRuntimeConfig()

// Fetch searches from API
const { data: searches, pending, error } = await useFetch(`${config.public.apiBase}/searches`)

// Format date helper
const formatDate = (dateString) => {
  const date = new Date(dateString)
  return date.toLocaleDateString('cs-CZ', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>
