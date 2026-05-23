// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

import { useEffect, useState } from "react";

import { listResources } from "../api/generic";

const TILES = ["vpc", "subnet", "security_group", "disk", "compute", "load_balancer", "node", "node_pool"] as const;

export default function Overview() {
  const [counts, setCounts] = useState<Record<string, number>>({});
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all(TILES.map((k) => listResources(k).then((items) => [k, items.length] as const)))
      .then((pairs) => setCounts(Object.fromEntries(pairs)))
      .catch((e: unknown) => setError(String(e)));
  }, []);

  return (
    <section>
      <h2>Overview</h2>
      {error && <p className="error">{error}</p>}
      <div className="cards">
        {TILES.map((k) => (
          <div className="card" key={k}>
            <div>{k}</div>
            <div className="value">{counts[k] ?? "-"}</div>
          </div>
        ))}
      </div>
    </section>
  );
}
