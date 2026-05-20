const API_BASE_URL = "http://localhost:8080";

export type ApiErrorResponse = {
  error?: string;
  message?: string;
};

export type AccountResponse = {
  account_id: string;
  status?: string;
  customer_id?: string;
  balance?: number;
  currency?: string;
};

export type TransferResponse = {
  transaction_id: string;
  status: string;
};

export type UserResponse = {
  user_id?: string;
  id?: string;
  email?: string;
  full_name?: string;
  status?: string;
};

export type LimitResponse = {
  status?: string;
  message?: string;
};

function createIdempotencyKey(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });

  const text = await response.text();

  let data: unknown = null;

  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }

  if (!response.ok) {
    const errorData = data as ApiErrorResponse;
    const message =
      errorData?.error ||
      errorData?.message ||
      response.statusText ||
      "Request failed";

    throw new Error(message);
  }

  return data as T;
}

/**
 * POST /api/v1/accounts
 */
export function createAccount(customerId: string, currency: string) {
  return request<AccountResponse>("/api/v1/accounts", {
    method: "POST",
    body: JSON.stringify({
      customer_id: customerId,
      currency,
      idempotency_key: createIdempotencyKey(),
    }),
  });
}

/**
 * GET /api/v1/accounts/{id}
 */
export function getAccount(accountId: string) {
  return request<AccountResponse>(
    `/api/v1/accounts/${encodeURIComponent(accountId)}`
  );
}

/**
 * POST /api/v1/accounts/{id}/balance
 */
export function updateBalance(
  accountId: string,
  amount: number,
  currency: string
) {
  return request<AccountResponse>(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/balance`,
    {
      method: "POST",
      body: JSON.stringify({
        amount,
        currency,
        idempotency_key: createIdempotencyKey(),
      }),
    }
  );
}

/**
 * POST /api/v1/accounts/{id}/freeze
 */
export function freezeAccount(accountId: string) {
  return request<{ status: string }>(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/freeze`,
    {
      method: "POST",
      body: JSON.stringify({
        idempotency_key: createIdempotencyKey(),
      }),
    }
  );
}

/**
 * PATCH /api/v1/accounts/{id}/status
 */
export function updateAccountStatus(accountId: string, status: string) {
  return request<AccountResponse>(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/status`,
    {
      method: "PATCH",
      body: JSON.stringify({
        status,
        idempotency_key: createIdempotencyKey(),
      }),
    }
  );
}

/**
 * DELETE /api/v1/accounts/{id}
 */
export function closeAccount(accountId: string) {
  return request<{ status: string }>(
    `/api/v1/accounts/${encodeURIComponent(accountId)}`,
    {
      method: "DELETE",
    }
  );
}

/**
 * GET /api/v1/accounts/{id}/history
 */
export function getAccountHistory(accountId: string) {
  return request<unknown>(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/history`
  );
}

/**
 * POST /api/v1/transfers
 */
export function transferFunds(
  sourceAccountId: string,
  destinationAccountId: string,
  amount: number,
  currency: string
) {
  return request<TransferResponse>("/api/v1/transfers", {
    method: "POST",
    body: JSON.stringify({
      source_account_id: sourceAccountId,
      destination_account_id: destinationAccountId,
      amount,
      currency,
      idempotency_key: createIdempotencyKey(),
    }),
  });
}

/**
 * GET /api/v1/transfers/{id}
 */
export function getTransferStatus(transactionId: string) {
  return request<TransferResponse>(
    `/api/v1/transfers/${encodeURIComponent(transactionId)}`
  );
}

/**
 * POST /api/v1/users/register
 */
export function registerUser(
  email: string,
  password: string,
  fullName: string
) {
  return request<UserResponse>("/api/v1/users/register", {
    method: "POST",
    body: JSON.stringify({
      email,
      password,
      full_name: fullName,
      idempotency_key: createIdempotencyKey(),
    }),
  });
}

/**
 * GET /api/v1/users/{id}
 */
export function getUserProfile(userId: string) {
  return request<UserResponse>(
    `/api/v1/users/${encodeURIComponent(userId)}`
  );
}

/**
 * POST /api/v1/analytics/limit
 */
export function setDailyLimit(
  accountId: string,
  amount: number,
  currency: string
) {
  return request<LimitResponse>("/api/v1/analytics/limit", {
    method: "POST",
    body: JSON.stringify({
      account_id: accountId,
      amount,
      currency,
      idempotency_key: createIdempotencyKey(),
    }),
  });
}