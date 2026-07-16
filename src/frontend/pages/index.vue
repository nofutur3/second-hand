<template>
  <main class="mx-auto max-w-3xl px-6 py-12">
    <div class="mb-10 flex items-end justify-between gap-4 border-b border-line pb-6">
      <div>
        <h1 class="font-serif text-3xl font-medium leading-tight text-ink">Saved searches</h1>
        <p class="mt-2 max-w-md text-[15px] leading-relaxed text-mute">
          Every keyword tracked across Bazos, Sbazar, Avizo, Inzeruj, Aukro, and eBay.
        </p>
      </div>
      <p v-if="searches?.length" class="shrink-0 font-mono text-sm text-faint">
        {{ searches.length }} tracked
      </p>
    </div>

    <div class="mb-10">
      <form class="flex items-center gap-3" @submit.prevent="createSearch">
        <input
          v-model="newKeyword"
          type="text"
          maxlength="255"
          placeholder="Track a new keyword&hellip;"
          aria-label="Keyword to track"
          class="flex-1 rounded-sm border border-line bg-surface px-3 py-2 font-mono text-sm text-ink placeholder:text-faint focus:border-stamp"
        />
        <button
          type="submit"
          :disabled="creating || !newKeyword.trim()"
          class="shrink-0 rounded-sm bg-stamp px-4 py-2 font-mono text-sm text-white transition-opacity hover:opacity-90 disabled:opacity-40"
        >
          {{ creating ? 'Tracking…' : 'Track' }}
        </button>
      </form>
      <p v-if="createError" class="mt-3 text-sm text-error">{{ createError }}</p>
    </div>

    <!-- Loading -->
    <div v-if="pending" class="space-y-5" aria-hidden="true">
      <div v-for="i in 4" :key="i" class="flex items-center justify-between py-3">
        <div class="h-4 w-40 animate-pulse rounded-sm bg-line"></div>
        <div class="h-3 w-24 animate-pulse rounded-sm bg-line"></div>
      </div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="border-l-2 border-error bg-error-dim px-5 py-4">
      <p class="font-medium text-error">Couldn't load saved searches</p>
      <p class="mt-1 text-[15px] text-ink/80">{{ error.message }}</p>
      <p class="mt-2 font-mono text-xs text-mute">Make sure the API is reachable at {{ config.public.apiBase }}</p>
    </div>

    <!-- Empty -->
    <div v-else-if="!searches || searches.length === 0" class="py-4">
      <p class="text-[15px] leading-relaxed text-mute">
        Nothing saved yet. Track a keyword above, or from the command line:
      </p>
      <pre class="mt-4 overflow-x-auto rounded-sm border border-line bg-surface px-4 py-3 font-mono text-sm text-ink">search -keyword="joy-con pair"</pre>
    </div>

    <!-- List -->
    <ul v-else>
      <li v-for="search in searches" :key="search.id" class="border-b border-line last:border-b-0">
        <NuxtLink
          :to="`/search/${search.id}`"
          class="group flex items-center justify-between gap-4 py-4 transition-colors hover:bg-surface"
        >
          <span class="font-serif text-lg text-ink transition-colors group-hover:text-stamp">
            {{ search.keyword }}
          </span>
          <span class="flex shrink-0 items-center gap-3 font-mono text-xs text-faint">
            {{ search.product_count }} offer{{ search.product_count === 1 ? '' : 's' }}
            <span class="text-line">&middot;</span>
            {{ formatDate(search.updated_at) }}
            <span class="text-stamp opacity-0 transition-opacity group-hover:opacity-100">&rarr;</span>
          </span>
        </NuxtLink>
      </li>
    </ul>
  </main>
</template>

<script setup>
const config = useRuntimeConfig()

const apiBase = useApiBase()

const { data: searches, pending, error, refresh } = await useFetch(`${apiBase}/searches`)

const newKeyword = ref('')
const creating = ref(false)
const createError = ref('')

const createSearch = async () => {
  const keyword = newKeyword.value.trim()
  if (!keyword) return

  creating.value = true
  createError.value = ''
  try {
    await $fetch(`${apiBase}/searches`, {
      method: 'POST',
      body: { keyword }
    })
    newKeyword.value = ''
    await refresh()
  } catch (e) {
    createError.value = e?.data?.message || e?.message || "Couldn't save this search."
  } finally {
    creating.value = false
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
</script>
