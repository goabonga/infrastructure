// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

import { useEffect, useState } from "react";

import { listACLs, listVPCs } from "../api/client";

export default function Overview() {
  const [vpcs, setVpcs] = useState<number | null>(null);
  const [acls, setAcls] = useState<number | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([listVPCs(), listACLs()])
      .then(([v, a]) => {
        setVpcs(v.length);
        setAcls(a.length);
      })
      .catch((e: unknown) => setError(String(e)));
  }, []);

  return (
    <section>
      <h2>Overview</h2>
      {error && <p className="error">{error}</p>}
      <div className="cards">
        <div className="card">
          <div>VPCs</div>
          <div className="value">{vpcs ?? "-"}</div>
        </div>
        <div className="card">
          <div>ACL policies</div>
          <div className="value">{acls ?? "-"}</div>
        </div>
      </div>
    </section>
  );
}
