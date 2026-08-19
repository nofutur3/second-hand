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

      <!-- Shop filter, remembered per search -->
      <div
        v-if="availableShops.length > 1"
        class="mt-4 flex flex-wrap items-center gap-2 font-mono text-xs uppercase tracking-wide text-mute"
      >
        <span>Shops</span>
        <button
          v-for="shop in availableShops"
          :key="shop"
          type="button"
          class="rounded-sm border px-2 py-0.5 normal-case tracking-normal transition-colors"
          :class="isShopSelected(shop) ? 'border-stamp text-stamp' : 'border-line text-faint'"
          @click="toggleShop(shop)"
        >
          {{ shop }}
        </button>
      </div>

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
                <template v-if="product.auction_type === 'auction'">
                  auction &middot; {{ product.bid_count }} bid{{ product.bid_count === 1 ? '' : 's' }}
                </template>
                <template v-else>sale</template>
                <template v-if="product.is_good_offer">
                  <span class="mx-1 text-line">&middot;</span>
                  <span class="text-stamp">good offer</span>
                </template>
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
                  <span class="text-error">{{ product.auction_type === 'auction' ? 'auction ended · last seen bid' : 'no longer listed' }}</span>
                </template>
                <template v-if="product.is_hidden">
                  <span class="mx-1 text-line">&middot;</span>
                  hidden
                </template>
                <template v-if="product.shop_source === 'ebay.com'">
                  <span class="mx-1 text-line">&middot;</span>
                  <button type="button" class="text-mute hover:text-stamp" @click="toggleDescription(product)">
                    {{ descriptionState(product).loading ? 'Loading…' : (descriptionState(product).expanded ? 'Hide description' : 'Description') }}
                  </button>
                </template>
                <span class="mx-1 text-line">&middot;</span>
                <button type="button" class="text-mute hover:text-stamp" @click="toggleHidden(product)">
                  {{ product.is_hidden ? 'Unhide' : 'Hide' }}
                </button>
              </p>
              <p
                v-if="descriptionState(product).expanded && descriptionState(product).text"
                class="mt-1.5 text-[14px] leading-relaxed text-mute"
              >
                {{ descriptionState(product).text }}
              </p>
              <p v-if="descriptionState(product).error" class="mt-1.5 text-[14px] text-error">
                {{ descriptionState(product).error }}
              </p>
            </div>
            <div class="shrink-0 whitespace-nowrap text-right">
              <template v-if="product.shipping_cost != null">
                <p class="font-mono text-xs text-faint">{{ formatPrice(product.price) }} {{ product.currency }}</p>
                <p class="font-mono text-xs text-faint">+ {{ formatPrice(product.shipping_cost) }} {{ product.currency }} shipping</p>
                <p class="font-mono text-lg font-medium text-tag">
                  = {{ formatPrice(product.price + product.shipping_cost) }} <span class="text-sm text-faint">{{ product.currency }}</span>
                </p>
              </template>
              <p v-else class="font-mono text-lg font-medium text-tag">
                {{ formatPrice(product.price) }} <span class="text-sm text-faint">{{ product.currency }}</span>
              </p>
            </div>
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
  const statusFiltered = showHidden.value ? products : products.filter((p) => !p.is_hidden && p.is_active)
  if (selectedShops.value === null) return statusFiltered
  return statusFiltered.filter((p) => selectedShops.value.includes(p.shop_source))
})

// Shop/adapter filter - remembered per search in localStorage. null means
// "not loaded from localStorage yet" and is treated as "show every shop",
// so SSR/initial render never has to guess at client-only storage.
const shopFilterKey = `snoopy:shopFilter:${searchId}`
const selectedShops = ref(null)

const availableShops = computed(() => {
  const products = data.value?.products ?? []
  return [...new Set(products.map((p) => p.shop_source))].sort()
})

const isShopSelected = (shop) => selectedShops.value === null || selectedShops.value.includes(shop)

const toggleShop = (shop) => {
  const current = selectedShops.value === null ? [...availableShops.value] : selectedShops.value
  selectedShops.value = current.includes(shop) ? current.filter((s) => s !== shop) : [...current, shop]
}

// Restoring from localStorage needs availableShops to actually be
// populated - a plain onMounted snapshot can fire before the client has
// the fetched product list attached (payload hydration doesn't always
// resolve synchronously before mount), leaving selectedShops stuck at
// "everything filtered out" forever. Watching (rather than a one-shot
// mount hook) re-runs this once real data shows up, whenever that is.
let restoredShopsFromStorage = false
watch(
  availableShops,
  (shops) => {
    if (restoredShopsFromStorage || shops.length === 0 || !import.meta.client) return
    restoredShopsFromStorage = true

    let stored = null
    try {
      stored = JSON.parse(localStorage.getItem(shopFilterKey))
    } catch {
      stored = null
    }
    if (Array.isArray(stored)) {
      const stillValid = stored.filter((s) => shops.includes(s))
      selectedShops.value = stillValid.length > 0 ? stillValid : [...shops]
    } else {
      selectedShops.value = [...shops]
    }
  },
  { immediate: true }
)

watch(selectedShops, (shops) => {
  if (shops !== null) localStorage.setItem(shopFilterKey, JSON.stringify(shops))
})

// eBay descriptions are fetched on demand (see cmd/api's /ebay-description)
// and cached here per product ID so re-toggling doesn't refetch.
const descriptionStates = reactive({})
const descriptionState = (product) =>
  descriptionStates[product.id] ?? { expanded: false, loading: false, text: '', error: '' }

const toggleDescription = async (product) => {
  if (!descriptionStates[product.id]) {
    descriptionStates[product.id] = { expanded: false, loading: false, text: '', error: '' }
  }
  // Read back through the reactive proxy (not the plain object literal
  // above) - mutating the raw object directly wouldn't trigger reactivity.
  const state = descriptionStates[product.id]

  if (state.expanded) {
    state.expanded = false
    return
  }
  state.expanded = true
  if (state.text || state.loading) return

  state.loading = true
  state.error = ''
  try {
    const res = await $fetch(`${apiBase}/ebay-description`, { query: { url: product.url } })
    state.text = res.description
  } catch (e) {
    state.error = e?.data?.message || e?.message || "Couldn't load description."
  } finally {
    state.loading = false
  }
}

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
