// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

import { afterEach, describe, expect, it, vi } from "vitest";

import { setToken } from "../auth";
import { createVPC, listVPCs } from "./client";

describe("api client", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it("lists VPCs and forwards the bearer token", async () => {
    setToken("tok");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        items: [{ metadata: { uid: "vpc-1", generation: 1, createdAt: "" }, spec: { cidr: "10.0.0.0/16" }, status: {} }],
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const vpcs = await listVPCs();
    expect(vpcs).toHaveLength(1);
    expect(vpcs[0].metadata.uid).toBe("vpc-1");

    const [url, opts] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/v1/vpc");
    expect((opts.headers as Record<string, string>)["Authorization"]).toBe("Bearer tok");
  });

  it("sends the spec on create", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ metadata: { uid: "vpc-1", generation: 1, createdAt: "" }, spec: { cidr: "10.0.0.0/16" }, status: {} }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await createVPC("vpc-1", { cidr: "10.0.0.0/16" });
    const [url, opts] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/v1/vpc/vpc-1");
    expect(opts.method).toBe("PUT");
    expect(JSON.parse(opts.body as string)).toEqual({ spec: { cidr: "10.0.0.0/16" } });
  });

  it("throws on a non-2xx response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 401 }));
    await expect(listVPCs()).rejects.toThrow();
  });
});
