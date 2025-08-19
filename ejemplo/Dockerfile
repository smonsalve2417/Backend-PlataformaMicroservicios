# Usar imagen base de Go
FROM golang:1.23

# Crear directorio de trabajo
WORKDIR /app

# Copiar todo el código al contenedor
COPY . .

# Construir el ejecutable
RUN go build -o app main.go

# Exponer puerto 8080
EXPOSE 8080

# Comando para ejecutar la app
CMD ["./app"]
