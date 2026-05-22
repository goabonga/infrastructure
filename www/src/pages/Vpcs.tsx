// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

import { type FormEvent, useEffect, useState } from "react";

import { type VPC, createVPC, deleteVPC, listVPCs } from "../api/client";

export default function Vpcs() {
  const [items, setItems] = useState<VPC[]>([]);
  const [uid, setUid] = useState("");
  const [cidr, setCidr] = useState("10.0.0.0/16");
  const [error, setError] = useState("");

  function reload() {
    listVPCs()
      .then(setItems)
      .catch((e: unknown) => setError(String(e)));
  }

  useEffect(reload, []);

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await createVPC(uid, { cidr });
      setUid("");
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  async function onDelete(id: string) {
    setError("");
    try {
      await deleteVPC(id);
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <section>
      <h2>VPCs</h2>
      {error && <p className="error">{error}</p>}

      <form className="row" onSubmit={onCreate}>
        <label>
          UID
          <br />
          <input value={uid} onChange={(e) => setUid(e.target.value)} placeholder="vpc-1" required />
        </label>
        <label>
          CIDR
          <br />
          <input value={cidr} onChange={(e) => setCidr(e.target.value)} required />
        </label>
        <button type="submit">Create</button>
      </form>

      <table>
        <thead>
          <tr>
            <th>UID</th>
            <th>CIDR</th>
            <th>Phase</th>
            <th>Bridge</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {items.map((v) => (
            <tr key={v.metadata.uid}>
              <td>{v.metadata.uid}</td>
              <td>{v.spec.cidr}</td>
              <td>{v.status.phase ?? "-"}</td>
              <td>{v.status.bridgeName ?? "-"}</td>
              <td>
                <button onClick={() => onDelete(v.metadata.uid)}>Delete</button>
              </td>
            </tr>
          ))}
          {items.length === 0 && (
            <tr>
              <td colSpan={5}>No VPCs.</td>
            </tr>
          )}
        </tbody>
      </table>
    </section>
  );
}
