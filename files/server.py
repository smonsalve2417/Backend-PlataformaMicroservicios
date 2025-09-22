import os
import json
import urllib.parse
from http.server import BaseHTTPRequestHandler, HTTPServer
from app import microservicio  # cada microservicio define su lógica aquí

# Nombre de la ruta configurable por variable de entorno
MICROSERVICIO_NAME = os.getenv("MICROSERVICIO_NAME", "default")


class RequestHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        # Obtener la ruta solicitada
        path = urllib.parse.urlparse(self.path).path.strip("/")

        # Validar que coincida con el microservicio
        if path != MICROSERVICIO_NAME:
            self.send_response(404)
            self.end_headers()
            self.wfile.write(json.dumps({
                "error": f"Ruta no valida. Use /{MICROSERVICIO_NAME}"
            }).encode("utf-8"))
            return

        # Leer body
        content_length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(content_length).decode("utf-8")

        try:
            body_json = json.loads(body)
        except json.JSONDecodeError:
            body_json = {"raw": body}

        # Construir request
        request_data = {
            "headers": dict(self.headers),
            "body": body_json
        }

        # Ejecutar el microservicio
        response_data = microservicio(request_data)

        # Responder
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(response_data).encode("utf-8"))


def run(port=8000):
    server_address = ("", port)
    httpd = HTTPServer(server_address, RequestHandler)
    print(f"Microservicio '{MICROSERVICIO_NAME}' corriendo en http://localhost:{port}/{MICROSERVICIO_NAME}")
    httpd.serve_forever()


if __name__ == "__main__":
    run()
