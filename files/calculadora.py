# app.py

from typing import Any, Dict

OPS_PERMITIDAS = {"sum", "sub", "mul", "div"}


def _to_float(value: Any, name: str) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        raise ValueError(f"El parámetro '{name}' debe ser numérico.")


def microservicio(request_json: Dict[str, Any]) -> Dict[str, Any]:
    try:
        body = (request_json or {}).get("body", {}) or {}

        op = str(body.get("op", "")).strip().lower()
        if op not in OPS_PERMITIDAS:
            return {
                "ok": False,
                "error": "Operación inválida. Usa: sum | sub | mul | div"
            }

        a = _to_float(body.get("a", None), "a")
        b = _to_float(body.get("b", None), "b")

        if op == "sum":
            result = a + b
        elif op == "sub":
            result = a - b
        elif op == "mul":
            result = a * b
        elif op == "div":
            if b == 0:
                return {"ok": False, "error": "División por cero no permitida."}
            result = a / b

        return {
            "ok": True,
            "op": op,
            "a": a,
            "b": b,
            "result": result
        }

    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        # Fallback de seguridad
        return {"ok": False, "error": f"Error inesperado: {e}"}
