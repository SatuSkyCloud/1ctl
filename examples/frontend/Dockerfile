FROM nginxinc/nginx-unprivileged:alpine
COPY --chown=101:101 --chmod=644 nginx.conf /etc/nginx/conf.d/default.conf
COPY --chown=101:101 --chmod=644 index.html /usr/share/nginx/html/
EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
