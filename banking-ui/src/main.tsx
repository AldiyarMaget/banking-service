import { FormEvent, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  createAccount,
  getAccount,
  transferFunds,
  type AccountResponse,
  type TransferResponse,
} from "./api";

import "./styles.css";

type ApiResult = AccountResponse | TransferResponse | Record<string, unknown>;

function App() {
  const [customerId, setCustomerId] = useState("");
  const [currency, setCurrency] = useState("KZT");

  const [accountId, setAccountId] = useState("");
  const [sourceAccountId, setSourceAccountId] = useState("");
  const [destinationAccountId, setDestinationAccountId] = useState("");
  const [amount, setAmount] = useState("");

  const [result, setResult] = useState<ApiResult | null>(null);
  const [error, setError] = useState("");

  async function runAction(action: () => Promise<ApiResult>) {
    setError("");
    setResult(null);

    try {
      const data = await action();
      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    }
  }

  function handleCreateAccount(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    void runAction(() => createAccount(customerId, currency));
  }

  function handleGetAccount(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    void runAction(() => getAccount(accountId));
  }

  function handleTransfer(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    void runAction(() =>
      transferFunds(
        sourceAccountId,
        destinationAccountId,
        Number(amount),
        currency
      )
    );
  }

  return (
    <main className="page">
      <section className="hero">
        <h1>Banking Service UI</h1>
        <p>
          Simple React + TypeScript frontend for the API Gateway on{" "}
          <code>localhost:8080</code>.
        </p>
      </section>

      <section className="grid">
        <form className="card" onSubmit={handleCreateAccount}>
          <h2>Create account</h2>

          <label>
            Customer ID
            <input
              value={customerId}
              onChange={(event) => setCustomerId(event.target.value)}
              placeholder="customer-123"
              required
            />
          </label>

          <label>
            Currency
            <input
              value={currency}
              onChange={(event) => setCurrency(event.target.value)}
              placeholder="KZT"
              required
            />
          </label>

          <button type="submit">Create account</button>
        </form>

        <form className="card" onSubmit={handleGetAccount}>
          <h2>Get account</h2>

          <label>
            Account ID
            <input
              value={accountId}
              onChange={(event) => setAccountId(event.target.value)}
              placeholder="account-id"
              required
            />
          </label>

          <button type="submit">Get account</button>
        </form>

        <form className="card" onSubmit={handleTransfer}>
          <h2>Transfer funds</h2>

          <label>
            Source account ID
            <input
              value={sourceAccountId}
              onChange={(event) => setSourceAccountId(event.target.value)}
              placeholder="source-account-id"
              required
            />
          </label>

          <label>
            Destination account ID
            <input
              value={destinationAccountId}
              onChange={(event) => setDestinationAccountId(event.target.value)}
              placeholder="destination-account-id"
              required
            />
          </label>

          <label>
            Amount
            <input
              value={amount}
              onChange={(event) => setAmount(event.target.value)}
              placeholder="1000"
              type="number"
              min="1"
              required
            />
          </label>

          <label>
            Currency
            <input
              value={currency}
              onChange={(event) => setCurrency(event.target.value)}
              placeholder="KZT"
              required
            />
          </label>

          <button type="submit">Transfer</button>
        </form>
      </section>

      <section className="output">
        <h2>Response</h2>

        {error && <pre className="error">{error}</pre>}

        {result && <pre>{JSON.stringify(result, null, 2)}</pre>}

        {!error && !result && (
          <p className="muted">Submit a form to see the API response.</p>
        )}
      </section>
    </main>
  );
}

createRoot(document.getElementById("root") as HTMLElement).render(<App />);