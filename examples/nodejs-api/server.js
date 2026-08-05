const http = require('node:http');

const port = Number(process.env.PORT || 3000);

const server = http.createServer((request, response) => {
  response.setHeader('content-type', 'application/json');

  if (request.url === '/health') {
    response.end(JSON.stringify({ status: 'ok', runtime: process.version }));
    return;
  }

  if (request.url === '/') {
    response.end(JSON.stringify({ message: 'Hello from SatuSky' }));
    return;
  }

  response.statusCode = 404;
  response.end(JSON.stringify({ error: 'not found' }));
});

server.listen(port, '0.0.0.0', () => {
  console.log(`listening on 0.0.0.0:${port}`);
});
