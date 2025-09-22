def microservicio(request_json):
    body = request_json.get("body", {})
    nombre = body.get("nombre", "Mundo")
    return {"mensaje": f"Hola {nombre}"}
