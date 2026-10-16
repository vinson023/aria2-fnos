import http.server
import time

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        total = 64 * 1024 * 1024
        self.send_response(200)
        self.send_header('Content-Length', str(total))
        self.end_headers()
        sent = 0
        chunk = b'x' * (256 * 1024)
        while sent < 2 * 1024 * 1024:
            self.wfile.write(chunk)
            sent += len(chunk)
        self.wfile.flush()
        time.sleep(1)
        self.connection.close()

    def log_message(self, *args):
        pass

http.server.HTTPServer(('127.0.0.1', 18086), Handler).serve_forever()