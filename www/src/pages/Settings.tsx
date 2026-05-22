// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

import { type FormEvent, useState } from "react";

import { getToken, setToken } from "../auth";

export default function Settings() {
  const [token, setTokenValue] = useState(getToken());
  const [saved, setSaved] = useState(false);

  function onSave(e: FormEvent) {
    e.preventDefault();
    setToken(token.trim());
    setSaved(true);
  }

  return (
    <section>
      <h2>Settings</h2>
      <p>
        The dashboard talks to the control plane through this server&apos;s <code>/api</code> proxy. Paste a bearer token
        issued by the identity provider (or a static API token); it is stored in your browser only.
      </p>
      <form className="row" onSubmit={onSave}>
        <label>
          API token
          <br />
          <input
            style={{ width: "24rem" }}
            value={token}
            onChange={(e) => {
              setTokenValue(e.target.value);
              setSaved(false);
            }}
            placeholder="eyJhbGciOiJFUzI1NiIs..."
          />
        </label>
        <button type="submit">Save</button>
      </form>
      {saved && <p>Saved.</p>}
    </section>
  );
}
