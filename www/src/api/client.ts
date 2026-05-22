// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

import { getToken } from "../auth";

export interface ObjectMeta {
  uid: string;
  name?: string;
  generation: number;
  createdAt: string;
}

export interface VPCSpec {
  cidr: string;
}

export interface VPCStatus {
  phase?: string;
  bridgeName?: string;
}

export interface VPC {
  metadata: ObjectMeta;
  spec: VPCSpec;
  status: VPCStatus;
}

export interface ACLRule {
  action: string;
  protocol?: string;
  port?: number;
  cidr?: string;
  rateLimit?: string;
}

export interface ACLPolicySpec {
  rules: ACLRule[];
}

export interface ACLPolicy {
  metadata: ObjectMeta;
  spec: ACLPolicySpec;
  status: { phase?: string };
}

interface List<T> {
  items: T[];
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  const resp = await fetch(`/api/v1${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (!resp.ok) {
    throw new Error(`${method} ${path}: ${resp.status}`);
  }
  if (resp.status === 204) {
    return undefined as T;
  }
  return (await resp.json()) as T;
}

export async function listVPCs(): Promise<VPC[]> {
  return (await request<List<VPC>>("GET", "/vpc")).items ?? [];
}

export async function createVPC(uid: string, spec: VPCSpec): Promise<VPC> {
  return request<VPC>("PUT", `/vpc/${uid}`, { spec });
}

export async function deleteVPC(uid: string): Promise<void> {
  await request<void>("DELETE", `/vpc/${uid}`);
}

export async function listACLs(): Promise<ACLPolicy[]> {
  return (await request<List<ACLPolicy>>("GET", "/acl_policy")).items ?? [];
}

export async function createACL(uid: string, spec: ACLPolicySpec): Promise<ACLPolicy> {
  return request<ACLPolicy>("PUT", `/acl_policy/${uid}`, { spec });
}

export async function deleteACL(uid: string): Promise<void> {
  await request<void>("DELETE", `/acl_policy/${uid}`);
}
