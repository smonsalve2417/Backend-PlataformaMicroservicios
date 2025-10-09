# app.py
# Microservicio: obtiene todos los datos de la columna "nombre" de una tabla
# y cuenta cuántos nombres distintos hay, usando el servicio de base de datos Uninorte.

import requests
from typing import Any, Dict, List

BASE_URL = "https://roble-api.openlab.uninorte.edu.co/database"


def _err(msg: str) -> Dict[str, Any]:
    return {"ok": False, "error": msg}


def _ok(data: Dict[str, Any]) -> Dict[str, Any]:
    return {"ok": True, **data}


def microservicio(request_json: Dict[str, Any]) -> Dict[str, Any]:
    """
    Espera en el body:
    {
      "dbName": "token_project_xyz",
      "tableName": "usuarios",
      "access_token": "TU_ACCESS_TOKEN"
    }
    """
    try:
        body = (request_json or {}).get("body", {}) or {}
        db_name = body.get("dbName")
        table_name = body.get("tableName")
        token = body.get("access_token")

        if not db_name or not table_name or not token:
            return _err("Faltan parámetros: dbName, tableName o access_token")

        # Construir URL completa
        url = f"{BASE_URL}/{db_name}/read"
        params = {"tableName": table_name}
        headers = {"Authorization": f"Bearer {token}"}

        # Llamada GET al servicio
        resp = requests.get(url, headers=headers, params=params, timeout=30)
        if resp.status_code != 200:
            try:
                msg = resp.json()
            except Exception:
                msg = resp.text
            return _err(f"Error del servicio ({resp.status_code}): {msg}")

        data = resp.json()
        if not isinstance(data, list):
            return _err("Respuesta inesperada: se esperaba una lista de registros")

        # Extraer columna 'nombre'
        nombres: List[str] = []
        for fila in data:
            if not isinstance(fila, dict):
                continue
            valor = fila.get("nombre")
            if valor is None:
                continue
            s = str(valor).strip()
            if s:
                nombres.append(s)

        # Calcular total y distintos
        total = len(nombres)
        distintos = len(set(nombres))

        return _ok({
            "column": "nombre",
            "names": nombres,
            "total": total,
            "distinct_count": distintos
        })

    except requests.Timeout:
        return _err("Timeout al consultar el servicio.")
    except requests.RequestException as e:
        return _err(f"Error de red: {e}")
    except Exception as e:
        return _err(f"Error inesperado: {e}")
