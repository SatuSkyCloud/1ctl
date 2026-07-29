FROM node:22-alpine

WORKDIR /app
ENV NODE_ENV=production

COPY --chown=node:node package.json server.js ./

EXPOSE 3000
USER node
CMD ["npm", "start"]
