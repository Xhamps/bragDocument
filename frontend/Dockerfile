FROM node:24-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
COPY packages/ui/package.json packages/ui/
COPY packages/app/package.json packages/app/
RUN npm ci
COPY . .
ARG VITE_API_URL
ARG VITE_SUPABASE_URL
ARG VITE_SUPABASE_PUBLISHABLE_KEY
RUN npm run build

FROM nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/packages/app/dist /usr/share/nginx/html
EXPOSE 80
