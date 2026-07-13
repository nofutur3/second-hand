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
      <!-- Back Button -->
      <NuxtLink
        to="/"
        class="inline-flex items-center px-4 py-2 mb-6 text-purple-600 bg-white border-2 border-purple-600 rounded-lg hover:bg-purple-600 hover:text-white transition-colors duration-200 font-medium"
      >
        <svg class="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
        Back to Searches
      </NuxtLink>

      <!-- Loading State -->
      <div v-if="pending" class="flex items-center justify-center py-12">
        <div class="text-center">
          <div class="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
          <p class="mt-4 text-gray-600 text-lg">Loading products...</p>
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
            <h3 class="text-lg font-medium text-red-800">Error loading products</h3>
            <p class="mt-2 text-red-700">{{ error.message }}</p>
            <p class="mt-2 text-sm text-red-600">Make sure the API server is running and the search exists</p>
          </div>
        </div>
      </div>

      <!-- Content -->
      <div v-else-if="data">
        <!-- Search Info Card -->
        <div class="bg-white rounded-xl shadow-lg p-6 mb-8">
          <h2 class="text-3xl font-bold text-purple-600 mb-4">{{ data.search.keyword }}</h2>
          <div class="flex flex-wrap gap-6 text-sm text-gray-600">
            <div class="flex items-center">
              <svg class="h-5 w-5 mr-2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
              <span><strong class="text-gray-900">Created:</strong> {{ formatDate(data.search.created_at) }}</span>
            </div>
            <div class="flex items-center">
              <svg class="h-5 w-5 mr-2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              <span><strong class="text-gray-900">Updated:</strong> {{ formatDate(data.search.updated_at) }}</span>
            </div>
            <div class="flex items-center">
              <svg class="h-5 w-5 mr-2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
              </svg>
              <span><strong class="text-gray-900">Total Products:</strong> {{ data.total }}</span>
            </div>
          </div>
        </div>

        <!-- Products Section -->
        <div v-if="data.products && data.products.length > 0">
          <div class="flex items-center justify-between mb-6">
            <h3 class="text-2xl font-bold text-gray-900">Products</h3>
            <span class="inline-flex items-center px-4 py-2 rounded-full text-sm font-medium bg-gradient-to-r from-purple-600 to-indigo-600 text-white">
              {{ data.total }} found
            </span>
          </div>

          <div class="space-y-4">
            <div
              v-for="product in data.products"
              :key="product.id"
              class="bg-white rounded-xl shadow-md hover:shadow-xl transition-shadow duration-300 overflow-hidden"
            >
              <div class="p-6">
                <!-- Product Header -->
                <div class="flex flex-col md:flex-row md:items-start md:justify-between gap-4 mb-4">
                  <div class="flex-1">
                    <h4 class="text-xl font-bold text-gray-900 mb-2 leading-tight">
                      <a
                        :href="product.url"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="text-purple-600 hover:text-purple-700 hover:underline"
                      >
                        {{ product.title }}
                      </a>
                    </h4>
                  </div>
                  <div class="flex-shrink-0">
                    <div class="inline-flex items-center px-4 py-2 rounded-full text-lg font-bold bg-gradient-to-r from-purple-600 to-indigo-600 text-white">
                      {{ formatPrice(product.price) }} {{ product.currency }}
                    </div>
                  </div>
                </div>

                <!-- Description -->
                <p v-if="product.description" class="text-gray-600 mb-4 line-clamp-2">
                  {{ product.description }}
                </p>

                <!-- Product Badges -->
                <div class="flex flex-wrap gap-2">
                  <!-- Shop Badge -->
                  <span class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                    <svg class="h-3 w-3 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 11-4 0 2 2 0 014 0z" />
                    </svg>
                    {{ product.shop_source }}
                  </span>

                  <!-- Condition Badge -->
                  <span
                    class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium"
                    :class="{
                      'bg-green-100 text-green-800': product.condition === 'new',
                      'bg-yellow-100 text-yellow-800': product.condition === 'used',
                      'bg-purple-100 text-purple-800': product.condition === 'refurbished',
                      'bg-gray-100 text-gray-800': product.condition === 'unknown'
                    }"
                  >
                    {{ formatCondition(product.condition) }}
                  </span>

                  <!-- Type Badge -->
                  <span
                    class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium"
                    :class="product.auction_type === 'auction' ? 'bg-pink-100 text-pink-800' : 'bg-indigo-100 text-indigo-800'"
                  >
                    {{ formatAuctionType(product.auction_type) }}
                  </span>

                  <!-- Location Badge -->
                  <span v-if="product.location" class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-teal-100 text-teal-800">
                    <svg class="h-3 w-3 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                    {{ product.location }}
                  </span>

                  <!-- Ending Time Badge -->
                  <span v-if="product.ending_time" class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800">
                    <svg class="h-3 w-3 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    Ends: {{ formatDate(product.ending_time) }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Empty Products State -->
        <div v-else class="bg-white rounded-xl shadow-lg p-12 text-center">
          <div class="mx-auto flex items-center justify-center h-20 w-20 rounded-full bg-gray-100 mb-6">
            <svg class="h-10 w-10 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
            </svg>
          </div>
          <h2 class="text-2xl font-bold text-gray-900 mb-2">No products found</h2>
          <p class="text-gray-600">This search didn't find any products yet.</p>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
const route = useRoute()
const config = useRuntimeConfig()

// Get search ID from route
const searchId = route.params.id

// Fetch search with products from API
const { data, pending, error } = await useFetch(
  `${config.public.apiBase}/searches/${searchId}/products`
)

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

// Format price helper
const formatPrice = (price) => {
  return new Intl.NumberFormat('cs-CZ', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2
  }).format(price)
}

// Format condition helper
const formatCondition = (condition) => {
  const conditions = {
    'new': '✨ New',
    'used': '♻️ Used',
    'refurbished': '🔧 Refurbished',
    'unknown': '❓ Unknown'
  }
  return conditions[condition] || condition
}

// Format auction type helper
const formatAuctionType = (type) => {
  return type === 'auction' ? '🔨 Auction' : '💰 Sale'
}
</script>
