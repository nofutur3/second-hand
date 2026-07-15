/**
 * SSR runs inside the Docker network and needs the internal `api` hostname;
 * the browser can't resolve that at all and needs a URL it can actually
 * reach. import.meta.server is resolved per-bundle (server vs client), so
 * this picks the right one regardless of whether a given fetch happens
 * during the initial server render or from a client-side interaction.
 */
export const useApiBase = () => {
  const config = useRuntimeConfig()
  return import.meta.server ? config.apiBaseServer : config.public.apiBase
}
