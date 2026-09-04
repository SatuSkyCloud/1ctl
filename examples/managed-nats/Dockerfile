FROM node:22-alpine
WORKDIR /app
COPY --chown=node:node --chmod=0644 package.json package-lock.json ./
RUN npm ci --omit=dev
COPY --chown=node:node --chmod=0644 server.mjs ./
EXPOSE 8080
# The platform runs containers with runAsNonRoot. Kubernetes cannot verify a
# named user is non-root, so the UID must be numeric or the container never
# starts (CreateContainerConfigError).
USER 1000
CMD ["npm", "start"]
