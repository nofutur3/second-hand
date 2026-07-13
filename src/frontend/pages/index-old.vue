<template>
  <div>
    <header class="header">
      <div class="container">
        <h1>🔍 Second-Hand Shop Scraper</h1>
        <p>Browse products from Czech second-hand marketplaces</p>
      </div>
    </header>

    <div class="container">
      <div v-if="pending" class="loading">
        Loading searches
      </div>

      <div v-else-if="error" class="error">
        <h3>❌ Error loading searches</h3>
        <p>{{ error.message }}</p>
        <p style="margin-top: 10px;">
          <small>Make sure the API server is running on http://localhost:8091</small>
        </p>
      </div>

      <div v-else-if="!searches || searches.length === 0" class="empty-state">
        <h2>📭 No searches yet</h2>
        <p>Start by running a search from the command line:</p>
        <p><code>./search -keyword="hemingway"</code></p>
        <p style="margin-top: 20px;">
          <small>Or with Docker:</small><br>
          <code>docker-compose exec api ./search -keyword="hemingway"</code>
        </p>
      </div>

      <div v-else>
        <h2 style="margin-bottom: 20px; color: #333;">
          Found {{ searches.length }} search{{ searches.length !== 1 ? 'es' : '' }}
        </h2>

        <div class="search-list">
          <NuxtLink
            v-for="search in searches"
            :key="search.id"
            :to="`/search/${search.id}`"
            class="search-card"
          >
            <h2>{{ search.keyword }}</h2>
            <div class="meta">
              <span class="date">
                {{ formatDate(search.created_at) }}
              </span>
              <span class="badge">View Products →</span>
            </div>
          </NuxtLink>
        </div>
      </div>
    </div>
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
