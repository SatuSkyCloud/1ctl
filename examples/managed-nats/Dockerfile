FROM node:22-alpine
WORKDIR /app
COPY --chown=node:node --chmod=0644 package.json package-lock.json ./
RUN npm ci --omit=dev
COPY --chown=node:node --chmod=0644 server.mjs ./
EXPOSE 8080
USER node
CMD ["npm", "start"]
