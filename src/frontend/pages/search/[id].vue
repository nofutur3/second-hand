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
      <div class="mt-6 flex items-start justify-between gap-4 border-b border-line pb-6">
        <div>
          <h1 class="font-serif text-3xl font-medium leading-tight text-ink">{{ data.search.keyword }}</h1>
          <p class="mt-2 font-mono text-xs text-faint">
            tracked since {{ formatDate(data.search.created_at) }}
            <span class="mx-1.5 text-line">&middot;</span>
            updated {{ formatDate(data.search.updated_at) }}
            <span class="mx-1.5 text-line">&middot;</span>
            {{ data.total }} listing{{ data.total === 1 ? '' : 's' }}
          </p>
        </div>
        <button
          type="button"
          :disabled="deleting"
          class="shrink-0 font-mono text-xs uppercase tracking-wide text-mute transition-colors hover:text-error disabled:opacity-40"
          @click="deleteSearch"
        >
          {{ deleting ? 'Removing…' : 'Stop tracking' }}
        </button>
      </div>
      <p v-if="deleteError" class="mt-4 text-sm text-error">{{ deleteError }}</p>

      <!-- Hidden/delisted toggle -->
      <label
        v-if="data.products && data.products.length > 0"
        class="mt-4 flex w-fit cursor-pointer select-none items-center gap-2 font-mono text-xs uppercase tracking-wide text-mute"
      >
        <input v-model="showHidden" type="checkbox" class="accent-stamp" />
        Show hidden &amp; delisted
        <span v-if="hiddenCount" class="normal-case tracking-normal text-faint">({{ hiddenCount }})</span>
      </label>

      <!-- Products -->
      <ul v-if="visibleProducts.length > 0" class="mt-2">
        <li v-for="(product, index) in visibleProducts" :key="product.id">
          <hr v-if="index > 0" class="divider-perforated" aria-hidden="true" />
          <div
            class="flex flex-col gap-2 py-5 sm:flex-row sm:items-start sm:justify-between sm:gap-6"
            :class="{ 'opacity-50': product.is_hidden || !product.is_active }"
          >
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
                <template v-if="!product.is_active">
                  <span class="mx-1 text-line">&middot;</span>
                  <span class="text-error">no longer listed</span>
                </template>
                <template v-if="product.is_hidden">
                  <span class="mx-1 text-line">&middot;</span>
                  hidden
                </template>
                <span class="mx-1 text-line">&middot;</span>
                <button type="button" class="text-mute hover:text-stamp" @click="toggleHidden(product)">
                  {{ product.is_hidden ? 'Unhide' : 'Hide' }}
                </button>
              </p>
            </div>
            <p class="shrink-0 whitespace-nowrap font-mono text-lg font-medium text-tag sm:text-right">
              {{ formatPrice(product.price) }} <span class="text-sm text-faint">{{ product.currency }}</span>
            </p>
          </div>
        </li>
      </ul>

      <!-- Empty: nothing found at all -->
      <p v-else-if="!data.products || data.products.length === 0" class="mt-8 text-[15px] leading-relaxed text-mute">
        No listings found yet for this search.
      </p>

      <!-- Empty: everything found so far is hidden or delisted -->
      <p v-else class="mt-8 text-[15px] leading-relaxed text-mute">
        Every listing found so far is hidden or no longer listed. Turn on
        "Show hidden &amp; delisted" above to see them.
      </p>

      <p v-if="hideError" class="mt-4 text-sm text-error">{{ hideError }}</p>
    </template>
  </main>
</template>

<script setup>
const route = useRoute()
const apiBase = useApiBase()

const searchId = route.params.id

const { data, pending, error, refresh } = await useFetch(
  `${apiBase}/searches/${searchId}/products`
)

const deleting = ref(false)
const deleteError = ref('')

const deleteSearch = async () => {
  if (!confirm(`Stop tracking "${data.value.search.keyword}"? Previously found listings stay in the catalog, but this search won't check for new ones again.`)) {
    return
  }

  deleting.value = true
  deleteError.value = ''
  try {
    await $fetch(`${apiBase}/searches/${searchId}`, { method: 'DELETE' })
    await navigateTo('/')
  } catch (e) {
    deleteError.value = e?.data?.message || e?.message || "Couldn't remove this search."
    deleting.value = false
  }
}

const showHidden = ref(false)
const hideError = ref('')

// Default view: hide anything marked irrelevant/incorrect by hand, and
// anything cron no longer finds when it re-checks (delisted/sold) - both
// stay in the database untouched, just out of the way until asked for.
const visibleProducts = computed(() => {
  const products = data.value?.products ?? []
  return showHidden.value ? products : products.filter((p) => !p.is_hidden && p.is_active)
})

const hiddenCount = computed(() => {
  const products = data.value?.products ?? []
  return products.filter((p) => p.is_hidden || !p.is_active).length
})

const toggleHidden = async (product) => {
  const hidden = !product.is_hidden
  hideError.value = ''
  try {
    await $fetch(`${apiBase}/searches/${searchId}/products/${product.id}`, {
      method: 'PATCH',
      body: { hidden }
    })
    // Mutating `product` in place isn't reliably reactive here (it's a
    // plain object nested inside useFetch's payload), so re-pull the
    // list instead of guessing at Vue's reactivity depth.
    await refresh()
  } catch (e) {
    hideError.value = e?.data?.message || e?.message || "Couldn't update this listing."
  }
}

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
