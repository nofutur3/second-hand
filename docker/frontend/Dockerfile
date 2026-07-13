# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy source code
COPY ../../src/frontend .

# Build the application
RUN npm run build

# Runtime stage
FROM node:20-alpine

WORKDIR /app

# Copy built application
COPY --from=builder /app/.output ./.output
COPY --from=builder /app/package*.json ./

# Set environment variables
ENV HOST=0.0.0.0
ENV PORT=8092
ENV NODE_ENV=production

# Expose port
EXPOSE 8092

# Start the application
CMD ["node", ".output/server/index.mjs"]
