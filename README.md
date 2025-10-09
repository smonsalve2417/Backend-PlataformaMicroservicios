
# Repositorio del Front

- https://github.com/Is-monp/Microservices_f
---

# 🧠 Descripción del Proyecto

La API de Gestión de Contenedores es un servicio REST desarrollado en Go (Golang) que permite a los usuarios crear, administrar y monitorear contenedores Docker de forma segura y personalizada, vinculados a sus cuentas mediante autenticación JWT.

El sistema integra Docker, MongoDB y Go para ofrecer un entorno de ejecución controlado donde cada usuario puede desplegar y gestionar sus propios servicios o aplicaciones, con seguimiento histórico y estadísticas de uso.

Está pensado como un backend modular y escalable, ideal para integrarse en plataformas que necesiten ofrecer a los usuarios:

- Ejecución de aplicaciones o entornos aislados en contenedores.

- Control sobre los recursos desplegados.

- Monitoreo en tiempo real del estado y uso histórico de contenedores.

# 🎯 Objetivos Principales

- Permitir a los usuarios autenticarse y gestionar sus propios contenedores Docker.

- Implementar un flujo seguro de creación, inicio, detención y eliminación de contenedores.

- Proporcionar visualización histórica de despliegues y fallos.

- Facilitar la automatización de creación de imágenes a partir de archivos cargados (por ejemplo, app.py).

- Mantener un diseño modular, limpio y fácilmente integrable con sistemas externos.

---

# Arquitectura

![Descripción de la imagen](./assets/arquitectura.png)

---

# 🐳 API de Gestión de Contenedores en Go

Servicio REST en **Go** para la administración de contenedores Docker asociados a usuarios autenticados.

Permite registrar usuarios, autenticarlos con JWT, crear imágenes, desplegar, detener y eliminar contenedores, y consultar su historial de uso.

---

## 🚀 Requisitos Previos

- Go 1.21 o superior

- Docker instalado y en ejecución

- MongoDB accesible (local o remoto)

- Directorio `./files` con los siguientes archivos base:

- `Dockerfile`

- `server.py`

---

## ⚙️ Configuración del Proyecto

Crea un archivo `.env` (opcional) o define las variables directamente en el código:

```bash

MONGO_URI=mongodb://localhost:27017

MONGO_DB=containersDB

AUTH_SERVICE_URL=https://tu-url-de-auth-service

PROJECT_ID=tu-proyecto-id

PORT=8080

```

Estructura recomendada del proyecto:

```

.

├── files/

│ ├── Dockerfile

│ └── server.py

├── workspace/

│ └── [nombre del contenedor generado]

├── handlers/

│ ├── auth.go

│ ├── containers.go

│ └── middleware.go

├── store/

│ └── mongo.go

├── utils/

│ ├── json.go

│ ├── jwt.go

│ └── validator.go

├── main.go

└── README.md

```

---

## ▶️ Ejecución

Compila y ejecuta el servidor:

```bash

go run main.go

```

Por defecto, el servicio se inicia en `http://localhost:8080`.

---

## 🔑 Autenticación

Los endpoints protegidos requieren un **token JWT**.

Primero registra un usuario y luego inicia sesión para obtener el `accessToken`.

---

## 🧩 Endpoints

### 🧍‍♂️ Registro de Usuario

**POST** `/auth/signup`

#### 📤 Request

```json
{
  "email": "usuario@example.com",

  "password": "123456",

  "name": "Sebastián"
}
```

#### 📥 Response

```json
"User created successfully"
```

---

### 🔐 Inicio de Sesión

**POST** `/auth/login`

#### 📤 Request

```json
{
  "email": "usuario@example.com",

  "password": "123456"
}
```

#### 📥 Response

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",

  "refreshToken": "eyJhbGciOiJIUzI1NiIs..."
}
```

---

### 🏗️ Crear Nuevo Contenedor

**POST** `/new/container` _(requiere JWT)_

#### 📤 Request

```json
{
  "image": "python-app",

  "type": "backend",

  "description": "Servicio backend Flask"
}
```

#### 📥 Response

```json
{
  "image": "python-app",

  "type": "backend",

  "description": "Servicio backend Flask"
}
```

---

### 🗑️ Eliminar Contenedor

**POST** `/remove/container` _(requiere JWT)_

#### 📤 Request

```json
{
  "image": "python-app"
}
```

#### 📥 Response

```json
{
  "image": "python-app"
}
```

---

### ⏹️ Detener Contenedor

**POST** `/stop/container` _(requiere JWT)_

#### 📤 Request

```json
{
  "image": "python-app"
}
```

#### 📥 Response

```json
{
  "image": "python-app"
}
```

---

### ▶️ Iniciar Contenedor

**POST** `/start/container` _(requiere JWT)_

#### 📤 Request

```json
{
  "image": "python-app"
}
```

#### 📥 Response

```json
{
  "image": "python-app"
}
```

---

### 🧱 Crear Imagen desde Archivos

**POST** `/new/image` _(requiere JWT, multipart/form-data)_

#### 📤 Request

Ejemplo con `curl`:

```bash

curl -X POST http://localhost:8080/new/image \

-H "Authorization: Bearer <TOKEN>" \

-F "name=python-app" \

-F "app=@./app.py"

```

#### 📥 Response

```json
{
  "image": "python-app:latest"
}
```

---

### 📋 Listar Contenedores del Usuario

**GET** `/containers/list` _(requiere JWT)_

#### 📥 Response

```json
{
  "containers": [
    {
      "userId": "12345",

      "containerName": "python-app",

      "status": true,

      "description": "Servicio backend Flask",

      "createdAt": "2025-10-08T10:00:00Z",

      "updatedAt": "2025-10-08T10:00:00Z",

      "type": "backend"
    }
  ],

  "count": 1
}
```

---

### 📜 Historial de Contenedores

**GET** `/containers/history` _(requiere JWT)_

#### 📥 Response

```json
{
  "containers": [
    {
      "userId": "12345",

      "containerName": "python-app",

      "status": true,

      "createdAt": "2025-10-08T10:00:00Z"
    },

    {
      "userId": "12345",

      "containerName": "python-app",

      "status": false,

      "createdAt": "2025-10-08T11:00:00Z"
    }
  ],

  "count": 2
}
```

---

### 📊 Historial para Gráficos (últimos 30 días)

**GET** `/containers/graphic` _(requiere JWT)_

#### 📥 Response

```json
{
  "labels": ["9 sept", "10 sept", "11 sept"],

  "deployments": [2, 1, 3],

  "errors": [0, 1, 0]
}
```

---

### ⏮️ Último Registro de Historial

**GET** `/containers/last`

#### 📥 Response

```json
{
  "userId": "12345",

  "containerName": "python-app",

  "status": false,

  "createdAt": "2025-10-08T11:00:00Z"
}
```

O si no existen registros:

```json
{
  "message": "no history records found"
}
```

---

## ⚙️ Notas Técnicas

- Los contenedores e imágenes se gestionan mediante **Docker SDK for Go**

- Los registros se almacenan en **MongoDB**

- La autenticación se maneja con **JWT** validado por `WithJWTAuth`

- Las entradas son validadas con **go-playground/validator**

- Los registros de historial permiten análisis gráfico de despliegues y fallos en los últimos 30 días

## 🧪 Ejemplo Rápido con `curl`

```bash

# Registro

curl -X POST http://localhost:8080/auth/signup \

-H "Content-Type: application/json" \

-d '{"email":"test@example.com","password":"123456","name":"User"}'



# Login

curl -X POST http://localhost:8080/auth/login \

-H "Content-Type: application/json" \

-d '{"email":"test@example.com","password":"123456"}'



# Crear contenedor

curl -X POST http://localhost:8080/new/container \

-H "Authorization: Bearer <TOKEN>" \

-d '{"image":"python-app","type":"backend","description":"Flask backend"}'

```

```

```
