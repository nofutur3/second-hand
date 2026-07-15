<template>
  <main class="mx-auto max-w-3xl px-6 py-12">
    <NuxtLink
      to="/"
      class="inline-flex items-center gap-1.5 font-mono text-xs uppercase tracking-wide text-mute transition-colors hover:text-stamp"
    >
      <span aria-hidden="true">&larr;</span> All searches
    </NuxtLink>

    <!-- Loading -->
    <div v-if="pending" class="mt-8 space-y-6" aria-hidden="true">
      <div class="h-8 w-56 animate-pulse rounded-sm bg-line"></div>
      <div v-for="i in 3" :key="i" class="flex items-center justify-between py-4">
        <div class="h-4 w-64 animate-pulse rounded-sm bg-line"></div>
        <div class="h-4 w-16 animate-pulse rounded-sm bg-line"></div>
      </div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="mt-8 border-l-2 border-error bg-error-dim px-5 py-4">
      <p class="font-medium text-error">Couldn't load this search</p>
      <p class="mt-1 text-[15px] text-ink/80">{{ error.message }}</p>
    </div>

    <template v-else-if="data">
      <div class="mt-6 border-b border-line pb-6">
        <h1 class="font-serif text-3xl font-medium leading-tight text-ink">{{ data.search.keyword }}</h1>
        <p class="mt-2 font-mono text-xs text-faint">
          tracked since {{ formatDate(data.search.created_at) }}
          <span class="mx-1.5 text-line">&middot;</span>
          updated {{ formatDate(data.search.updated_at) }}
          <span class="mx-1.5 text-line">&middot;</span>
          {{ data.total }} listing{{ data.total === 1 ? '' : 's' }}
        </p>
      </div>

      <!-- Products -->
      <ul v-if="data.products && data.products.length > 0" class="mt-2">
        <li v-for="(product, index) in data.products" :key="product.id">
          <hr v-if="index > 0" class="divider-perforated" aria-hidden="true" />
          <div class="flex flex-col gap-2 py-5 sm:flex-row sm:items-start sm:justify-between sm:gap-6">
            <div class="min-w-0">
              <a
                :href="product.url"
                target="_blank"
                rel="noopener noreferrer"
                class="font-medium leading-snug text-ink hover:text-stamp"
              >
                {{ product.title }}
                <span class="ml-1 text-faint" aria-hidden="true">&#8599;</span>
              </a>
              <p v-if="product.description" class="mt-1.5 line-clamp-2 text-[14px] leading-relaxed text-mute">
                {{ product.description }}
              </p>
              <p class="mt-2 font-mono text-xs text-faint">
                {{ product.shop_source }}
                <span class="mx-1 text-line">&middot;</span>
                {{ formatCondition(product.condition) }}
                <span class="mx-1 text-line">&middot;</span>
                {{ product.auction_type === 'auction' ? 'auction' : 'sale' }}
                <template v-if="product.location">
                  <span class="mx-1 text-line">&middot;</span>
                  {{ product.location }}
                </template>
                <template v-if="product.ending_time">
                  <span class="mx-1 text-line">&middot;</span>
                  ends {{ formatDate(product.ending_time) }}
                </template>
              </p>
            </div>
            <p class="shrink-0 whitespace-nowrap font-mono text-lg font-medium text-tag sm:text-right">
              {{ formatPrice(product.price) }} <span class="text-sm text-faint">{{ product.currency }}</span>
            </p>
          </div>
        </li>
      </ul>

      <!-- Empty products -->
      <p v-else class="mt-8 text-[15px] leading-relaxed text-mute">
        No listings found yet for this search.
      </p>
    </template>
  </main>
</template>

<script setup>
const route = useRoute()
const config = useRuntimeConfig()

const searchId = route.params.id

const { data, pending, error } = await useFetch(
  `${config.public.apiBase}/searches/${searchId}/products`
)

const formatDate = (dateString) => {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-GB', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const formatPrice = (price) => {
  return new Intl.NumberFormat('en-GB', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2
  }).format(price)
}

const conditionLabels = {
  new: 'New',
  used: 'Used',
  like_new: 'Like new',
  good: 'Good',
  fair: 'Fair',
  poor: 'Poor',
  damaged: 'Damaged',
  unknown: 'Condition unknown'
}

const formatCondition = (condition) => conditionLabels[condition] || condition
</script>
