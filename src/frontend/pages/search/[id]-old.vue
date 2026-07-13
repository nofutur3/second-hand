<template>
  <div>
    <header class="header">
      <div class="container">
        <h1>🔍 Second-Hand Shop Scraper</h1>
        <p>Browse products from Czech second-hand marketplaces</p>
      </div>
    </header>

    <div class="container">
      <NuxtLink to="/" class="back-button">
        ← Back to Searches
      </NuxtLink>

      <div v-if="pending" class="loading">
        Loading products
      </div>

      <div v-else-if="error" class="error">
        <h3>❌ Error loading products</h3>
        <p>{{ error.message }}</p>
        <p style="margin-top: 10px;">
          <small>Make sure the API server is running and the search exists</small>
        </p>
      </div>

      <div v-else-if="data">
        <div class="search-details">
          <h2>{{ data.search.keyword }}</h2>
          <div class="info">
            <div class="info-item">
              <strong>Created:</strong>
              <span>{{ formatDate(data.search.created_at) }}</span>
            </div>
            <div class="info-item">
              <strong>Updated:</strong>
              <span>{{ formatDate(data.search.updated_at) }}</span>
            </div>
            <div class="info-item">
              <strong>Total Products:</strong>
              <span>{{ data.total }}</span>
            </div>
          </div>
        </div>

        <div v-if="data.products && data.products.length > 0">
          <div class="products-header">
            <h3>Products</h3>
            <span class="product-count">{{ data.total }} found</span>
          </div>

          <div class="product-list">
            <div
              v-for="product in data.products"
              :key="product.id"
              class="product-card"
            >
              <div class="product-header">
                <div class="product-title">
                  <h4>
                    <a :href="product.url" target="_blank" rel="noopener noreferrer">
                      {{ product.title }}
                    </a>
                  </h4>
                </div>
                <div class="product-price">
                  {{ formatPrice(product.price) }} {{ product.currency }}
                </div>
              </div>

              <p v-if="product.description" class="product-description">
                {{ product.description }}
              </p>

              <div class="product-meta">
                <span class="product-badge badge-shop">
                  🏪 {{ product.shop_source }}
                </span>

                <span
                  class="product-badge badge-condition"
                  :class="product.condition"
                >
                  {{ formatCondition(product.condition) }}
                </span>

                <span class="product-badge badge-type">
                  {{ formatAuctionType(product.auction_type) }}
                </span>

                <span v-if="product.location" class="product-badge badge-location">
                  📍 {{ product.location }}
                </span>

                <span v-if="product.ending_time" class="product-badge" style="background: #ffebee; color: #c62828;">
                  ⏰ Ends: {{ formatDate(product.ending_time) }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="empty-state" style="margin-top: 30px;">
          <h2>📭 No products found</h2>
          <p>This search didn't find any products yet.</p>
        </div>
      </div>
    </div>
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
