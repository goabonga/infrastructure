// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

import { apiRequest as request } from "./http";

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
