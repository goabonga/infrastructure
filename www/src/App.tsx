// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

import { Route, Routes } from "react-router-dom";

import Layout from "./components/Layout";
import Acls from "./pages/Acls";
import Overview from "./pages/Overview";
import Settings from "./pages/Settings";
import Vpcs from "./pages/Vpcs";

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Overview />} />
        <Route path="vpcs" element={<Vpcs />} />
        <Route path="acls" element={<Acls />} />
        <Route path="settings" element={<Settings />} />
      </Route>
    </Routes>
  );
}
