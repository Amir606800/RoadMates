FROM node:22.15.0-alpine3.21 AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_APP_VERSION=dev
ARG VITE_APP_COMMIT=unknown
ENV VITE_APP_VERSION=${VITE_APP_VERSION} VITE_APP_COMMIT=${VITE_APP_COMMIT}
RUN npm run build

FROM build AS test
RUN npm test

FROM nginxinc/nginx-unprivileged:1.28.0-alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
USER 101
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1:8080/version.json || exit 1
