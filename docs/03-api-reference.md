# API Reference (Panduan Integrasi Agen)

Dokumen ini ditujukan untuk **Agen/Klien** yang ingin mengintegrasikan sistem mereka (Laravel, PHP, Node.js, Python, dll) dengan Backend Distributor Pulsa.

- **Base URL:** `http://localhost:8080` (Ganti dengan domain Anda saat production, misal: `https://api.pulsa-anda.com`)
- **Format Data:** `application/json`
- **Autentikasi:** JWT (JSON Web Token)

---

## 🔐 1. Autentikasi (Login)

Setiap request ke endpoint (selain login) **wajib** menyertakan token yang didapat dari proses login.

### A. Login Agen

- **Method:** `POST`
- **Endpoint:** `/api/agent/login`
- **Body Request:**

  ```json
  {
    "username": "agent01",
    "password": "agent123"
  }
  ```

  {
  "success": true,
  "message": "login success",
  "data": {
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOjEsInJvbGUiOiJhZ2VudCIs..."
  }
  }

  {
  "success": false,
  "error": "invalid credentials"
  }
