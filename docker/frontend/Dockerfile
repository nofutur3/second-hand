# syntax=docker/dockerfile:1

# Shared base: dependency cache. Both the dev and build stages branch
# from here so `npm ci` only ever runs once.
FROM node:22-alpine AS base
WORKDIR /app
COPY src/frontend/package.json src/frontend/package-lock.json ./
RUN npm ci

# ---- development ----
# No source copied in - bind-mount src/frontend at /app at runtime (see
# compose.yaml's `frontend-dev` service) and this runs the Nuxt dev
# server with HMR against it.
FROM base AS dev
ENV HOST=0.0.0.0
EXPOSE 8092
CMD ["npm", "run", "dev", "--", "--port", "8092", "--host", "0.0.0.0"]

# ---- build ----
FROM base AS build
COPY src/frontend .
RUN npm run build

# ---- production ----
# Nitro's node-server preset (Nuxt's default) bundles the server's own
# dependencies into .output, so the runtime image needs nothing but
# Node itself and that directory - no node_modules, no package manager,
# no shell.
FROM gcr.io/distroless/nodejs22-debian12 AS production
WORKDIR /app
COPY --from=build /app/.output ./.output
ENV HOST=0.0.0.0
ENV PORT=8092
ENV NODE_ENV=production
EXPOSE 8092
CMD [".output/server/index.mjs"]
