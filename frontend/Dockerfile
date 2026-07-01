FROM node:22-slim AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN mkdir -p /src/public
ARG NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/v1
ENV NEXT_PUBLIC_API_BASE_URL=${NEXT_PUBLIC_API_BASE_URL}
RUN npm run build

FROM node:22-slim
WORKDIR /app
ENV NODE_ENV=production
COPY --from=build /src/package.json /src/package-lock.json ./
RUN npm ci --omit=dev
COPY --from=build /src/.next ./.next
COPY --from=build /src/public ./public
COPY --from=build /src/next.config.ts ./
EXPOSE 3000
CMD ["npm", "run", "start"]
