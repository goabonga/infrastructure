// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

import { type FormEvent, useEffect, useState } from "react";

import { type GenericResource, KINDS, createResource, deleteResource, listResources } from "../api/generic";
import { Fields, PhaseBadge, STATUS_SKIP } from "../components/Fields";

export default function Resources() {
  const [kind, setKind] = useState<string>("vpc");
  const [items, setItems] = useState<GenericResource[]>([]);
  const [uid, setUid] = useState("");
  const [specText, setSpecText] = useState("{}");
  const [error, setError] = useState("");

  function reload(k: string) {
    listResources(k)
      .then(setItems)
      .catch((e: unknown) => setError(String(e)));
  }

  useEffect(() => {
    setError("");
    reload(kind);
  }, [kind]);

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    setError("");
    let spec: Record<string, unknown>;
    try {
      spec = JSON.parse(specText) as Record<string, unknown>;
    } catch {
      setError("spec is not valid JSON");
      return;
    }
    try {
      await createResource(kind, uid, spec);
      setUid("");
      reload(kind);
    } catch (e) {
      setError(String(e));
    }
  }

  async function onDelete(id: string) {
    setError("");
    try {
      await deleteResource(kind, id);
      reload(kind);
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <section>
      <h2>Resources</h2>
      <p>
        Browse and manage any resource kind. Enter the spec as JSON, for example{" "}
        <code>{`{"vpcId":"...","cidr":"10.0.1.0/24","type":"public"}`}</code> for a subnet.
      </p>

      <label>
        Kind&nbsp;
        <select value={kind} onChange={(e) => setKind(e.target.value)}>
          {KINDS.map((k) => (
            <option key={k} value={k}>
              {k}
            </option>
          ))}
        </select>
      </label>

      {error && <p className="error">{error}</p>}

      <form className="row" onSubmit={onCreate}>
        <label>
          UID
          <br />
          <input value={uid} onChange={(e) => setUid(e.target.value)} placeholder={`${kind}-1`} required />
        </label>
        <label style={{ flex: 1 }}>
          Spec (JSON)
          <br />
          <textarea
            style={{ width: "100%", minWidth: "24rem", fontFamily: "monospace" }}
            rows={3}
            value={specText}
            onChange={(e) => setSpecText(e.target.value)}
          />
        </label>
        <button type="submit">Create</button>
      </form>

      <table>
        <thead>
          <tr>
            <th>UID</th>
            <th>Phase</th>
            <th>Spec</th>
            <th>Status</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {items.map((r) => (
            <tr key={r.metadata.uid}>
              <td>{r.metadata.uid}</td>
              <td>
                <PhaseBadge phase={r.status.phase} />
              </td>
              <td>
                <Fields data={r.spec} />
              </td>
              <td>
                <Fields data={r.status} skip={STATUS_SKIP} />
              </td>
              <td>
                <button onClick={() => onDelete(r.metadata.uid)}>Delete</button>
              </td>
            </tr>
          ))}
          {items.length === 0 && (
            <tr>
              <td colSpan={5}>No {kind} resources.</td>
            </tr>
          )}
        </tbody>
      </table>
    </section>
  );
}
