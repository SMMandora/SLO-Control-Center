FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
# VITE_BFF_URL is baked at build time; defaults to the published BFF port.
ARG VITE_BFF_URL=http://localhost:9090
ENV VITE_BFF_URL=$VITE_BFF_URL
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
