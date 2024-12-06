FROM node:22.4.1-alpine3.20 AS build
RUN npm install -g pnpm
WORKDIR /app
# cache dependencies
COPY package.json ./
RUN pnpm install
# full build
COPY . .
RUN pnpm build

FROM node:22.4.1-alpine3.20
COPY --from=build /app/node_modules /app/node_modules
COPY --from=build /app/dist /app/dist
ENTRYPOINT [ "node", "/app/dist/index.js" ]
