// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

import { NavLink, Outlet } from "react-router-dom";

import { getToken } from "../auth";

function navClass({ isActive }: { isActive: boolean }): string {
  return isActive ? "active" : "";
}

export default function Layout() {
  const hasToken = getToken() !== "";
  return (
    <div className="layout">
      <aside className="sidebar">
        <h1>infra</h1>
        <nav>
          <NavLink to="/" end className={navClass}>
            Overview
          </NavLink>
          <NavLink to="/vpcs" className={navClass}>
            VPCs
          </NavLink>
          <NavLink to="/acls" className={navClass}>
            ACL policies
          </NavLink>
          <NavLink to="/settings" className={navClass}>
            Settings
          </NavLink>
        </nav>
      </aside>
      <main className="content">
        {!hasToken && (
          <div className="banner">No API token set. Add one under Settings to talk to the control plane.</div>
        )}
        <Outlet />
      </main>
    </div>
  );
}
