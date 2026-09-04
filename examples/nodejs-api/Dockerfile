FROM node:22-alpine

WORKDIR /app
ENV NODE_ENV=production

COPY --chown=node:node package.json server.js ./

EXPOSE 3000
# The platform runs containers with runAsNonRoot. Kubernetes cannot verify a
# named user is non-root, so the UID must be numeric or the container never
# starts (CreateContainerConfigError).
USER 1000
CMD ["npm", "start"]
