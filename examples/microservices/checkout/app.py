import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.error import URLError
from urllib.request import urlopen


class Handler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:
        status = 200
        if self.path == "/health":
            body = {"status": "ok", "service": "checkout"}
        elif self.path == "/order-preview":
            try:
                with urlopen(
                    os.environ["CATALOG_URL"] + "/item", timeout=5
                ) as response:
                    body = {"checkout": "ok", "catalog": json.load(response)}
            except (KeyError, TimeoutError, URLError):
                status = 503
                body = {"checkout": "degraded", "catalog": "unavailable"}
        else:
            self.send_error(404)
            return

        encoded = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)


ThreadingHTTPServer(("0.0.0.0", 8000), Handler).serve_forever()
