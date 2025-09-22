# Imagen base oficial de Python
FROM python:3.11-slim

# Crear directorio de trabajo
WORKDIR /app

# Copiar los archivos del proyecto al contenedor
COPY server.py app.py ./

# Exponer el puerto donde correrá el servidor
EXPOSE 8000

# Comando por defecto para iniciar el servidor
CMD ["python", "server.py"]
