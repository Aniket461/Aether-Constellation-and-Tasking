FROM node:24-bookworm AS build
WORKDIR /app
ENV NG_CLI_ANALYTICS=false
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npx ng build --configuration=production

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist/frontend/browser /usr/share/nginx/html
EXPOSE 80
