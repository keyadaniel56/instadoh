# InstaDoh — Instant Cross-Border Lightning Payments

**InstaDoh** is a fintech platform that enables instant, low-cost cross-border payments between **Kenya** and **Uganda** by bridging **mobile money** (M-Pesa, MTN Mobile Money, Airtel Money) with the **Bitcoin Lightning Network**. Users can deposit local currency, send money across borders in seconds, and recipients receive funds directly to their mobile money wallets — all powered by Lightning's instant settlement and near-zero fees.

---

## Table of Contents

- [How It Works](#how-it-works)
- [Architecture Overview](#architecture-overview)
- [Tech Stack](#tech-stack)
- [API Endpoints](#api-endpoints)
  - [Health](#health)
  - [Authentication](#authentication)
  - [Users](#users)
  - [Payments (Lightning)](#payments-lightning)
  - [Cross-Border Transfers](#cross-border-transfers)
  - [M-Pesa (Kenya)](#m-pesa-kenya)
  - [Uganda Mobile Money](#uganda-mobile-money)
- [Data Models](#data-models)
- [Configuration](#configuration)
- [Getting Started](#getting-started)
- [Frontend](#frontend)

---

## How It Works

1. **Onboard** — Users sign up with their country (KE or UG) and link their mobile money number.
2. **Deposit** — Deposit KES via M-Pesa or UGX via MTN/Airtel Mobile Money into their InstaDoh wallet.
3. **Send** — Initiate a cross-border transfer. InstaDoh converts the amount at the current exchange rate, deducts a small fee, and settles the transfer via the Lightning Network.
4. **Receive** — The recipient's mobile money wallet is credited in their local currency (KES in Kenya, UGX in Uganda) instantly.

The Lightning Network acts as the settlement layer between the two mobile money corridors, enabling instant finality and near-zero transaction costs compared to traditional bank transfers or remittance services.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    React Frontend (Vite)                     │
│  Landing Page | Dashboard | Login/Signup | Cross-Border UI  │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP (REST API)
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                  Go Backend (Gin Framework)                  │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐ │
│  │  Auth    │  │ Payments │  │Cross-    │  │ Mobile     │ │
│  │ Handlers │  │ Handlers │  │Border    │  │ Money      │ │
│  │          │  │          │  │Handlers  │  │ Handlers   │ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └─────┬──────┘ │
│       │              │             │               │        │
│  ┌────┴──────────────┴─────────────┴───────────────┴──────┐ │
│  │                    Services Layer                        │ │
│  │  PaymentService | CrossBorderService | ExchangeService  │ │
│  │  MpesaService | UgandaMobileService | LNDService        │ │
│  └──────────────────────────┬──────────────────────────────┘ │
│                             │                                │
│  ┌──────────────────────────┴──────────────────────────────┐ │
│  │                    Database (GORM)                       │ │
│  │         SQLite (dev) / PostgreSQL (production)          │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
   ┌──────────┐      ┌──────────────┐      ┌──────────────┐
   │   LND    │      │  Exchange    │      │  Mobile      │
   │ (Lightning│      │  Rate API    │      │  Money APIs  │
   │  Node)   │      │              │      │  (M-Pesa,    │
   │          │      │              │      │  MTN/Airtel) │
   └──────────┘      └──────────────┘      └──────────────┘
```

---

## Tech Stack

### Backend
- **Language:** Go
- **Framework:** Gin (HTTP router + middleware)
- **Database:** GORM (SQLite for development, PostgreSQL for production)
- **Lightning Network:** LND (gRPC client)
- **Authentication:** JWT (bcrypt password hashing)
- **Exchange Rates:** exchangerate-api.com (with in-memory caching)

### Frontend
- **Framework:** React 18
- **Build Tool:** Vite
- **Routing:** React Router v6
- **HTTP Client:** Axios
- **Styling:** Custom CSS

---

## API Endpoints

All API routes are prefixed with `/api/v1`. Protected routes require a valid JWT token in the `Authorization: Bearer <token>` header.

### Health

| Method | Path       | Auth | Description                     |
|--------|------------|------|---------------------------------|
| GET    | `/health`  | No   | Health check & API version info |

**Response:**
```json
{
  "status": "ok",
  "version": "1.0.0",
  "name": "InstaDoh API"
}
```

---

### Authentication

| Method | Path                    | Auth | Description                        |
|--------|-------------------------|------|------------------------------------|
| POST   | `/api/v1/auth/register` | No   | Register a new user                |
| POST   | `/api/v1/auth/login`    | No   | Login and receive JWT token        |
| POST   | `/api/v1/auth/refresh`  | Yes  | Refresh an existing JWT token      |

#### `POST /api/v1/auth/register`

**Request:**
```json
{
  "email": "user@example.com",
  "phone": "+254712345678",
  "password": "securepassword123",
  "full_name": "John Doe",
  "country_code": "KE",
  "role": "user"
}
```
- `country_code` must be a supported 2-letter ISO code (e.g., `KE`, `UG`).
- `role` is optional; defaults to `"user"`. Can be `"user"` or `"merchant"`.
- `password` must be at least 8 characters.

**Response (201 Created):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "phone": "+254712345678",
    "full_name": "John Doe",
    "country_code": "KE",
    "currency": "KES",
    "balance": 0,
    "role": "user",
    "created_at": "2026-06-18T00:00:00Z"
  }
}
```

#### `POST /api/v1/auth/login`

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": { "...same as register response..." }
}
```

#### `POST /api/v1/auth/refresh`

**Request:** Requires `Authorization: Bearer <token>` header.

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...new-token..."
}
```

---

### Users

| Method | Path                  | Auth | Description                |
|--------|-----------------------|------|----------------------------|
| GET    | `/api/v1/countries`   | No   | List all supported countries |
| GET    | `/api/v1/users/me`    | Yes  | Get current user's profile   |

#### `GET /api/v1/countries`

**Response (200 OK):**
```json
[
  {
    "code": "KE",
    "name": "Kenya",
    "currency": "KES",
    "currency_name": "Kenyan Shilling",
    "flag": "🇰🇪",
    "is_active": true
  },
  {
    "code": "UG",
    "name": "Uganda",
    "currency": "UGX",
    "currency_name": "Ugandan Shilling",
    "flag": "🇺🇬",
    "is_active": true
  }
]
```

#### `GET /api/v1/users/me`

**Response (200 OK):**
```json
{
  "id": 1,
  "email": "user@example.com",
  "phone": "+254712345678",
  "full_name": "John Doe",
  "country_code": "KE",
  "currency": "KES",
  "balance": 5000.00,
  "role": "user",
  "created_at": "2026-06-18T00:00:00Z"
}
```

---

### Payments (Lightning)

All payment endpoints require authentication.

| Method | Path                                    | Auth | Description                          |
|--------|-----------------------------------------|------|--------------------------------------|
| POST   | `/api/v1/payments/invoices`             | Yes  | Create a Lightning invoice (receive) |
| POST   | `/api/v1/payments/send`                 | Yes  | Send a Lightning payment             |
| GET    | `/api/v1/payments/transactions`         | Yes  | List user's transactions (paginated) |
| GET    | `/api/v1/payments/transactions/:id`     | Yes  | Get a specific transaction           |
| GET    | `/api/v1/payments/balance`              | Yes  | Get user's wallet balance            |
| GET    | `/api/v1/payments/stats`                | Yes  | Get user's payment statistics        |
| POST   | `/api/v1/payments/convert`              | Yes  | Convert between currencies           |
| GET    | `/api/v1/payments/rates`                | No   | Get current exchange rates           |
| POST   | `/api/v1/payments/webhook`              | No   | LND webhook callback (no auth)       |

#### `POST /api/v1/payments/invoices`

Create a Lightning invoice to receive funds.

**Request:**
```json
{
  "amount": 100.00,
  "currency": "USD",
  "description": "Payment for services",
  "expiry_seconds": 3600
}
```

**Response (201 Created):**
```json
{
  "id": 1,
  "payment_request": "lnbc1...",
  "amount": 100.00,
  "currency": "USD",
  "description": "Payment for services",
  "status": "pending",
  "expires_at": "2026-06-18T01:00:00Z",
  "created_at": "2026-06-18T00:00:00Z"
}
```

#### `POST /api/v1/payments/send`

Send a Lightning payment by providing a BOLT11 invoice.

**Request:**
```json
{
  "invoice": "lnbc1...",
  "amount": 50.00,
  "currency": "USD"
}
```

**Response (200 OK):**
```json
{
  "id": 2,
  "user_id": 1,
  "amount": 50.00,
  "currency": "USD",
  "amount_btc": 5000000,
  "direction": "outgoing",
  "status": "completed",
  "payment_hash": "abc123...",
  "payment_request": "lnbc1...",
  "counterparty": "recipient@domain.com",
  "description": "",
  "created_at": "2026-06-18T00:05:00Z",
  "completed_at": "2026-06-18T00:05:01Z"
}
```

#### `GET /api/v1/payments/transactions`

**Query Parameters:**
- `page` (default: `1`)
- `limit` (default: `20`, max: `100`)

**Response (200 OK):**
```json
{
  "data": [ { "...transaction..." } ],
  "total": 42,
  "page": 1,
  "limit": 20,
  "totalPages": 3
}
```

#### `GET /api/v1/payments/transactions/:id`

**Response (200 OK):** Single transaction object.

#### `GET /api/v1/payments/balance`

**Response (200 OK):**
```json
{
  "currency": "KES",
  "balance": 5000.00
}
```

#### `GET /api/v1/payments/stats`

**Response (200 OK):**
```json
{
  "total_transactions": 42,
  "total_sent": 15000.00,
  "total_received": 20000.00,
  "average_amount": 833.33
}
```

#### `POST /api/v1/payments/convert`

Convert an amount between any two supported currencies.

**Request:**
```json
{
  "amount": 1000,
  "from_currency": "KES",
  "to_currency": "UGX"
}
```

**Response (200 OK):**
```json
{
  "amount": 1000,
  "from_currency": "KES",
  "to_currency": "UGX",
  "result": 33000.00
}
```

#### `GET /api/v1/payments/rates`

**Query Parameters:**
- `currency` (default: `"USD"`)

**Response (200 OK):**
```json
{
  "currency": "KES",
  "rate_to_usd": 0.0075,
  "btc_usd": 65000.00
}
```

#### `POST /api/v1/payments/webhook`

LND webhook callback for invoice settlement notifications.

**Request:**
```json
{
  "payment_hash": "abc123...",
  "status": "settled",
  "preimage": "def456...",
  "settled_amt": 5000000
}
```

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

---

### Cross-Border Transfers

All cross-border endpoints require authentication.

| Method | Path                                              | Auth | Description                              |
|--------|---------------------------------------------------|------|------------------------------------------|
| GET    | `/api/v1/cross-border/quote`                      | Yes  | Get a quote for a cross-border transfer  |
| POST   | `/api/v1/cross-border/send`                       | Yes  | Initiate a cross-border transfer         |
| GET    | `/api/v1/cross-border/transactions`               | Yes  | List cross-border transactions (paginated) |
| GET    | `/api/v1/cross-border/transactions/:id`           | Yes  | Get a specific cross-border transaction  |

#### `GET /api/v1/cross-border/quote`

**Query Parameters:**
- `from_currency` (required, e.g., `KES`)
- `to_currency` (required, e.g., `UGX`)
- `amount` (required, must be > 0)

**Response (200 OK):**
```json
{
  "from_currency": "KES",
  "to_currency": "UGX",
  "send_amount": 1000,
  "receive_amount": 32670,
  "exchange_rate": 33.0,
  "fee": 30,
  "total_in_fiat": 1000,
  "valid_until": "2026-06-18T00:05:00Z"
}
```

#### `POST /api/v1/cross-border/send`

**Request:**
```json
{
  "recipient_phone": "+256712345678",
  "recipient_country": "UG",
  "amount": 1000,
  "currency": "KES",
  "description": "Family support"
}
```

**Response (200 OK):**
```json
{
  "transaction": {
    "id": 1,
    "sender_id": 1,
    "receiver_phone": "+256712345678",
    "receiver_country": "UG",
    "send_amount": 1000,
    "send_currency": "KES",
    "receive_amount": 32670,
    "receive_currency": "UGX",
    "exchange_rate": 33.0,
    "fee": 30,
    "status": "completed",
    "lightning_tx_id": "abc123...",
    "description": "Family support",
    "created_at": "2026-06-18T00:10:00Z",
    "completed_at": "2026-06-18T00:10:02Z"
  },
  "message": "Cross-border transfer initiated successfully"
}
```

#### `GET /api/v1/cross-border/transactions`

**Query Parameters:**
- `page` (default: `1`)
- `limit` (default: `20`, max: `100`)

**Response (200 OK):**
```json
{
  "data": [ { "...cross-border transaction..." } ],
  "total": 10,
  "page": 1,
  "limit": 20,
  "totalPages": 1
}
```

#### `GET /api/v1/cross-border/transactions/:id`

**Response (200 OK):** Single cross-border transaction object.

---

### M-Pesa (Kenya)

M-Pesa endpoints integrate with Safaricom's Daraja API for STK Push (deposits) and B2C (withdrawals).

| Method | Path                          | Auth | Description                              |
|--------|-------------------------------|------|------------------------------------------|
| POST   | `/api/v1/mpesa/deposit`       | Yes  | Initiate M-Pesa STK Push deposit         |
| POST   | `/api/v1/mpesa/withdraw`      | Yes  | Initiate M-Pesa B2C withdrawal           |
| POST   | `/api/v1/mpesa/callback`      | No   | M-Pesa STK Push callback from Safaricom  |
| POST   | `/api/v1/mpesa/result`        | No   | M-Pesa B2C result callback               |
| POST   | `/api/v1/mpesa/timeout`       | No   | M-Pesa timeout callback                  |

#### `POST /api/v1/mpesa/deposit`

Initiate an STK Push to the user's phone for deposit confirmation.

**Request:**
```json
{
  "phone_number": "254712345678",
  "amount": 1000
}
```

**Response (200 OK):**
```json
{
  "checkout_request_id": "ws_CO_1806202600001234",
  "response_code": "0",
  "response_description": "Success. Request accepted for processing",
  "merchant_request_id": "12345-67890-12345",
  "amount": 1000,
  "phone_number": "254712345678",
  "status": "pending"
}
```

#### `POST /api/v1/mpesa/withdraw`

Initiate a B2C (Business-to-Customer) withdrawal to the user's M-Pesa.

**Request:**
```json
{
  "phone_number": "254712345678",
  "amount": 500
}
```

**Response (200 OK):** Similar to deposit response.

#### `POST /api/v1/mpesa/callback`

Safaricom's callback endpoint for STK Push results. The system automatically credits the user's wallet on successful payment.

#### `POST /api/v1/mpesa/result`

Safaricom's callback endpoint for B2C withdrawal results.

#### `POST /api/v1/mpesa/timeout`

Handles M-Pesa transaction timeouts. Always returns success to Safaricom.

---

### Uganda Mobile Money

| Method | Path                                    | Auth | Description                                    |
|--------|-----------------------------------------|------|------------------------------------------------|
| POST   | `/api/v1/uganda-mobile/deposit`         | Yes  | Initiate Uganda mobile money deposit           |
| POST   | `/api/v1/uganda-mobile/withdraw`        | Yes  | Initiate Uganda mobile money withdrawal        |
| POST   | `/api/v1/uganda-mobile/callback`        | No   | Uganda mobile money callback from provider     |

#### `POST /api/v1/uganda-mobile/deposit`

**Request:**
```json
{
  "phone_number": "256712345678",
  "amount": 50000,
  "provider": "mtn"
}
```
- `provider` can be `"mtn"` or `"airtel"`.

**Response (200 OK):**
```json
{
  "status": "pending",
  "reference": "txn_ref_12345",
  "message": "Deposit initiated. Check your phone to complete."
}
```

#### `POST /api/v1/uganda-mobile/withdraw`

**Request:** Same structure as deposit.

**Response (200 OK):** Similar to deposit response.

#### `POST /api/v1/uganda-mobile/callback`

Provider callback for transaction status updates. On successful deposit, the user's wallet is credited.

---

## Data Models

### User
| Field        | Type     | Description                          |
|-------------|----------|--------------------------------------|
| id          | uint     | Primary key                          |
| email       | string   | Unique email address                 |
| phone       | string   | Unique phone number                  |
| password    | string   | bcrypt-hashed password               |
| full_name   | string   | User's full name                     |
| country_code| string   | ISO 2-letter country code (KE, UG)   |
| currency    | string   | Default currency (KES, UGX, USD)     |
| balance     | float64  | Wallet balance                       |
| role        | enum     | `user`, `merchant`, `admin`          |
| ln_address  | string   | Lightning address (optional)         |
| is_active   | bool     | Account active status                |

### Transaction
| Field          | Type     | Description                          |
|----------------|----------|--------------------------------------|
| id             | uint     | Primary key                          |
| user_id        | uint     | Foreign key to users                 |
| amount         | float64  | Amount in fiat                       |
| currency       | string   | Fiat currency code                   |
| amount_btc     | int64    | Amount in millisatoshis              |
| direction      | enum     | `incoming` or `outgoing`             |
| status         | enum     | `pending`, `completed`, `failed`, `expired` |
| payment_hash   | string   | Lightning payment hash               |
| payment_request| string   | BOLT11 invoice string                |
| preimage       | string   | Payment preimage (on settlement)     |
| counterparty   | string   | Counterparty identifier              |
| description    | string   | Transaction description              |
| fee_msat       | int64    | Routing fee in millisatoshis         |
| exchange_rate  | float64  | Rate at time of transaction          |

### CrossBorderTransaction
| Field            | Type     | Description                          |
|------------------|----------|--------------------------------------|
| id               | uint     | Primary key                          |
| sender_id        | uint     | Foreign key to users (sender)        |
| receiver_phone   | string   | Recipient's phone number             |
| receiver_country | string   | Recipient's country (KE or UG)       |
| send_amount      | float64  | Amount sent in sender's currency     |
| send_currency    | string   | Sender's currency                    |
| receive_amount   | float64  | Amount received in recipient's currency |
| receive_currency | string   | Recipient's currency                 |
| exchange_rate    | float64  | Exchange rate applied                |
| fee              | float64  | Service fee                          |
| status           | string   | `pending`, `completed`, `failed`     |
| lightning_tx_id  | string   | Lightning payment hash               |
| description      | string   | Transfer description                 |

### MobileMoneyTransaction
| Field         | Type     | Description                          |
|---------------|----------|--------------------------------------|
| id            | uint     | Primary key                          |
| user_id       | uint     | Foreign key to users                 |
| type          | string   | `deposit` or `withdrawal`            |
| provider      | enum     | `mpesa` or `uganda_mobile`           |
| provider_ref  | string   | Provider's transaction reference     |
| phone_number  | string   | User's phone number                  |
| amount        | float64  | Transaction amount                   |
| currency      | string   | Currency code                        |
| status        | enum     | `pending`, `completed`, `failed`     |
| failure_reason| string   | Reason for failure (if any)          |

---

## Configuration

Configuration is managed via environment variables (loaded from a `.env` file). See [`.env.example`](instadoh-backend/.env.example) for all options.

| Variable                  | Default                        | Description                          |
|---------------------------|--------------------------------|--------------------------------------|
| `SERVER_HOST`             | `0.0.0.0`                     | Server bind address                  |
| `SERVER_PORT`             | `8080`                         | Server port                          |
| `DB_DRIVER`               | `sqlite`                       | Database driver (`sqlite`/`postgres`)|
| `DB_PATH`                 | `instadoh.db`                  | SQLite file path                     |
| `JWT_SECRET`              | `change-me-in-production`      | JWT signing secret                   |
| `JWT_EXPIRATION_HOURS`    | `24`                           | Token expiration in hours            |
| `LND_HOST`                | `localhost:10009`              | LND gRPC host                        |
| `LND_TLS_CERT`            | `/root/.lnd/tls.cert`          | LND TLS certificate path             |
| `LND_MACAROON`            | `/root/.lnd/admin.macaroon`    | LND admin macaroon path              |
| `EXCHANGE_API_KEY`        | `""`                           | Exchange rate API key                |
| `EXCHANGE_BASE_URL`       | `https://api.exchangerate-api.com/v4/latest` | Exchange rate API base URL |
| `MPESA_CONSUMER_KEY`      | `""`                           | M-Pesa Daraja API consumer key       |
| `MPESA_CONSUMER_SECRET`   | `""`                           | M-Pesa Daraja API consumer secret    |
| `MPESA_SHORTCODE`         | `174379`                       | M-Pesa business shortcode            |
| `MPESA_ENVIRONMENT`       | `sandbox`                      | M-Pesa environment (`sandbox`/`production`) |
| `UG_MOBILE_API_BASE_URL`  | `https://api.uganda-mobile.co.ug/v1` | Uganda mobile API base URL     |

---

## Getting Started

### Prerequisites

- Go 1.21+
- Node.js 18+
- LND node (optional for development — the app runs in limited mode without it)

### Backend

```bash
# Clone the repository
git clone https://github.com/keyadaniel56/instadoh.git
cd instadoh/instadoh-backend

# Copy and configure environment
cp .env.example .env
# Edit .env with your configuration

# Run with SQLite (development)
go run .

# Or build and run
go build -o instadoh .
./instadoh
```

The server starts on `http://0.0.0.0:8080` by default.

### Frontend

```bash
cd instadoh/instadoh-frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

The frontend runs on `http://localhost:5173` by default and proxies API requests to the backend.

### Docker

```bash
cd instadoh/instadoh-backend
docker-compose up
```

---

## Frontend

The React frontend provides:

- **Landing Page** — Hero section, features overview, how it works, pricing, and call-to-action
- **Authentication** — Login and signup forms with JWT-based session management
- **Dashboard** — Protected route showing wallet balance, transaction history, and cross-border transfer interface
- **Cross-Border Transfer** — Form to send money between Kenya and Uganda with real-time exchange rate quotes

### Frontend API Client Structure

| File                          | Description                          |
|-------------------------------|--------------------------------------|
| `src/api/client.js`           | Axios instance with JWT interceptor  |
| `src/api/auth.js`             | Auth API calls (login, register)     |
| `src/api/payments.js`         | Payment API calls                    |
| `src/api/crossBorder.js`      | Cross-border transfer API calls      |
| `src/context/AuthContext.jsx` | React context for auth state         |
| `src/components/ProtectedRoute.jsx` | Route guard for authenticated pages |

---

## License

MIT