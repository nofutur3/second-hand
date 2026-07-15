<template>
  <div class="min-h-screen bg-paper text-ink">
    <header class="border-b border-line">
      <div class="mx-auto max-w-3xl px-6 py-4">
        <NuxtLink
          to="/"
          class="inline-flex items-baseline gap-2 font-mono text-[13px] font-medium uppercase tracking-[0.2em] text-ink hover:text-stamp"
        >
          Second Hand
          <span class="font-sans text-[11px] normal-case tracking-normal text-faint">market watch</span>
        </NuxtLink>
      </div>
    </header>

    <slot />
  </div>
</template>
