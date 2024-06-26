<a name="readme-top"></a>

[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![LinkedIn][linkedin-shield]][linkedin-url]

<br />
<div align="center">
  <a href="https://github.com/conceptcodes/uas-go">
    <!-- REPLACE WITH HEADER -->
    <!-- <img src="public/logo.svg" alt="Logo" width="80" height="80"> -->
  </a>

<h1 align="center">User Authentication Microservice </h1>
  <p align="center">
    <a href="https://github.com/conceptcodes/uas-go/issues/new?assignees=&labels=&projects=&template=bug_report.md&title=">Report Bug</a>
    ·
    <a href="https://github.com/conceptcodes/uas-go/issues/new?assignees=&labels=&projects=&template=feature_request.md&title=">Request Feature</a>
  </p>
</div>

<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#features">Features</a></li>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li>
      <a href="#usage">Usage</a>
    </li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
  </ol>
</details>

## About The Project

This Golang microservice offers secure user authentication for your applications. It supports email/password login and can be extended to include additional methods like OTP or magic link login.

<p align="right">(<a href="#readme-top">back to top</a>)</p>


### Features

- User Registration/Login with Email and Password
- Email/Password Login
- Secure Password Hashing (bcrypt)
- JSON Web Token (JWT) based Authentication


### Built With

- [![Bcrypt][bcrypt-shield]][bcrypt-url]
- [![Docker][Docker]][docker-url]
- [![Golang][Golang]][golang-url]
- [![Json Web Token][JWT]][jwt-url]
- [![MySQL][mysql-shield]][mysql-url]


<p align="right">(<a href="#readme-top">back to top</a>)</p>


## Getting Started

To get a local copy up and running follow these simple steps. 

### Prerequisites

- Docker
- Golang 1.21 or higher
- MySQL 5.7 or higher

### Installation

1. Clone the repo
  ```sh
  git clone https://github.com/conceptcodes/uas-go.git
  ```

2. Install the dependencies and create an `.env` file in the root directory. Copy the contents of the `.env.example` file and replace the values with your own.
  ```sh
  go mod download 
  cp .env.example .env
  ```

3. Run the migrations
  ```sh
  make migrate
  ```

4. Start the server
  ```sh
  make start
  ```

5. The server should now be running on `http://localhost:8080`

<p align="right">(<a href="#readme-top">back to top</a>)</p>


## Usage


**Health Check**

```sh
curl -X GET \
  https://localhost:8080/api/v1/health
```
```json
{
  "status": "ok"
}
```

---

**Onboard a new Tenant**

> This action will add an authorization header to the response. All subsequent requests must include a base64-encoded authorization header. This header value is generated by combining your department ID and service secret, separated by a colon. This allows the service to authenticate the tenant and authorize requests.

```sh
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "departmentName": "Department Name",
    "departmentId": "c4c2fab4-0a4f-4f8d-924c-611aa4af2fe2"
  }' \
  https://localhost:8080/api/v1/tenants
```
```json
{
  "id": "",
  "departmentName": "Department Name",
  "departmentId": "826dad3c-ae6d-4603-8190-730cad295035",
}
```

---

**Register a new User**

```sh
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "strong_password",
    "name": "John Smith"
  }' \
  https://localhost:8080/api/v1/users/credential/register
```
```json
{
  "id": "",
  "name": "John Smith",
  "email": "user@example.com"
}
```

---

**Login (Credentials)**

```sh
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "strong_password"
  }' \
  https://localhost:8080/api/v1/users/credential/login
```
```json
{
  "accessToken": "eyJhbGciNiIsInR5C..." (JWT token string),
  "refreshToken": "eyJhbGciNiIsInR4C..." 
}
```

---

**Login (OTP)**

```sh
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "phoneNumber": ""
  }' \
  https://localhost:8080/api/v1/users/otp/send
```
```json
{
  "message": "OTP sent successfully"
}
```

---

**Verify OTP**

```sh
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "phoneNumber": "",
    "otp": ""
  }' \
  https://localhost:8080/api/v1/users/otp/verify
```
```json
{
  "accessToken": "eyJhbGciNiIsInR5C...",
  "refreshToken": "eyJhbGciNiIsInR5C..."
}
```

---

**Refresh Token**

```sh
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Cookie: <access_token>" \
  https://localhost:8080/api/v1/users/refresh-token
```

### Security Considerations

- HTTPS for all communication.
- Rate limiting for login attempts.
- RBAC for user roles and permissions.


## Roadmap
- [x] Add support for email verification
- [x] Add support for password reset
- [x] Add support for rate limiting
- [x] Add support for OTP login
- [ ] Add support for magic link login
- [x] Add support for RBAC
- [ ] Add support for audit logging



[contributors-shield]: https://img.shields.io/github/contributors/conceptcodes/uas-go.svg?style=for-the-badge
[contributors-url]: https://github.com/conceptcodes/uas-go/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/conceptcodes/uas-go.svg?style=for-the-badge
[forks-url]: https://github.com/conceptcodes/uas-go/network/members
[stars-shield]: https://img.shields.io/github/stars/conceptcodes/uas-go.svg?style=for-the-badge
[stars-url]: https://github.com/conceptcodes/uas-go/stargazers
[issues-shield]: https://img.shields.io/github/issues/conceptcodes/uas-go.svg?style=for-the-badge
[issues-url]: https://github.com/conceptcodes/uas-go/issues
[linkedin-shield]: https://img.shields.io/badge/-LinkedIn-black.svg?style=for-the-badge&logo=linkedin&colorB=555
[linkedin-url]: https://linkedin.com/in/david-ojo-66a12a147
[Golang]: https://img.shields.io/badge/-Golang-00ADD8?style=for-the-badge&logo=go&logoColor=white
[golang-url]: https://golang.org/
[Docker]: https://img.shields.io/badge/-Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white
[docker-url]: https://www.docker.com/
[JWT]: https://img.shields.io/badge/-JWT-000000?style=for-the-badge&logo=json-web-tokens&logoColor=white
[jwt-url]: https://jwt.io/
[bcrypt-shield]: https://img.shields.io/badge/-Bcrypt-00599C?style=for-the-badge&logo=bcrypt&logoColor=white
[bcrypt-url]: https://www.npmjs.com/package/bcrypt
[mysql-shield]: https://img.shields.io/badge/-MySQL-4479A1?style=for-the-badge&logo=mysql&logoColor=white
[mysql-url]: https://www.mysql.com/


