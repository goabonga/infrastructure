// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

import { type FormEvent, useEffect, useState } from "react";

import { type ACLPolicy, createACL, deleteACL, listACLs } from "../api/client";

export default function Acls() {
  const [items, setItems] = useState<ACLPolicy[]>([]);
  const [uid, setUid] = useState("");
  const [action, setAction] = useState("allow");
  const [protocol, setProtocol] = useState("tcp");
  const [port, setPort] = useState("443");
  const [error, setError] = useState("");

  function reload() {
    listACLs()
      .then(setItems)
      .catch((e: unknown) => setError(String(e)));
  }

  useEffect(reload, []);

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      const portNum = Number(port);
      await createACL(uid, {
        rules: [{ action, protocol, port: Number.isFinite(portNum) && portNum > 0 ? portNum : undefined }],
      });
      setUid("");
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  async function onDelete(id: string) {
    setError("");
    try {
      await deleteACL(id);
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <section>
      <h2>ACL policies</h2>
      {error && <p className="error">{error}</p>}

      <form className="row" onSubmit={onCreate}>
        <label>
          UID
          <br />
          <input value={uid} onChange={(e) => setUid(e.target.value)} placeholder="web" required />
        </label>
        <label>
          Action
          <br />
          <select value={action} onChange={(e) => setAction(e.target.value)}>
            <option value="allow">allow</option>
            <option value="deny">deny</option>
          </select>
        </label>
        <label>
          Protocol
          <br />
          <select value={protocol} onChange={(e) => setProtocol(e.target.value)}>
            <option value="tcp">tcp</option>
            <option value="udp">udp</option>
            <option value="icmp">icmp</option>
            <option value="all">all</option>
          </select>
        </label>
        <label>
          Port
          <br />
          <input value={port} onChange={(e) => setPort(e.target.value)} />
        </label>
        <button type="submit">Create</button>
      </form>

      <table>
        <thead>
          <tr>
            <th>UID</th>
            <th>Rules</th>
            <th>Phase</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {items.map((p) => (
            <tr key={p.metadata.uid}>
              <td>{p.metadata.uid}</td>
              <td>{p.spec.rules.length}</td>
              <td>{p.status.phase ?? "-"}</td>
              <td>
                <button onClick={() => onDelete(p.metadata.uid)}>Delete</button>
              </td>
            </tr>
          ))}
          {items.length === 0 && (
            <tr>
              <td colSpan={4}>No ACL policies.</td>
            </tr>
          )}
        </tbody>
      </table>
    </section>
  );
}
