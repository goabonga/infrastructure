{{/*
SPDX-License-Identifier: MIT
Copyright (c) 2026 Chris <goabonga@pm.me>
*/}}

{{- define "infra.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "infra.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "infra.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Labels shared by every object. Kept in one place so a relabelling cannot
half-apply and leave selectors matching nothing.
*/}}
{{- define "infra.labels" -}}
helm.sh/chart: {{ include "infra.chart" . }}
app.kubernetes.io/name: {{ include "infra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: infra
{{- end -}}

{{- define "infra.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "infra.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
Refuse to render a replicated component with no shared state backend.

The file-backed store lives in a per-pod emptyDir, so two api replicas
without a DSN each serve a different view of the cluster and each believe they
are right. That is worse than not deploying: it looks healthy. Fail at
template time, where the message is readable, rather than at runtime where it
surfaces as resources that intermittently vanish.
*/}}
{{- define "infra.validateState" -}}
{{- $hasDSN := or .Values.state.dsn .Values.state.existingSecret -}}
{{- if not $hasDSN -}}
{{- range $name, $c := .Values.components -}}
{{- if and $c.enabled $c.needsState (gt (int $c.replicas) 1) -}}
{{- fail (printf "components.%s has replicas=%d and needs shared state, but neither state.dsn nor state.existingSecret is set. The file-backed store is a per-pod emptyDir and cannot be shared: set a PostgreSQL DSN, or drop this component to a single replica for development." $name (int $c.replicas)) -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Refuse to render infra-idp with no clients configured.

The daemon calls auth.ParseTokens on GOA_IDP_CLIENTS and exits non-zero when
it yields nothing, so an idp deployed without this does not come up degraded -
it crash-loops. Catch it at template time, where the message says what to set.
*/}}
{{- define "infra.validateIDP" -}}
{{- $idp := .Values.components.idp -}}
{{- if $idp.enabled -}}
{{- if not (or $idp.clients $idp.existingSecret) -}}
{{- fail "components.idp is enabled but neither components.idp.clients nor components.idp.existingSecret is set. infra-idp exits at startup with no clients to issue tokens to: set clients to \"token:subject\" pairs, point existingSecret at a secret holding them, or disable the component." -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Environment for a component: the state backend, then any caller-supplied
entries. The DSN comes from a secret reference whenever one is configured, so
the value never appears in the rendered manifest.
*/}}
{{- define "infra.stateEnv" -}}
{{- $root := .root -}}
{{- if $root.Values.state.existingSecret }}
- name: GOA_STATE_DSN
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.state.existingSecret }}
      key: {{ $root.Values.state.existingSecretKey }}
{{- else if $root.Values.state.dsn }}
- name: GOA_STATE_DSN
  value: {{ $root.Values.state.dsn | quote }}
{{- else }}
- name: GOA_STATE_DIR
  value: {{ $root.Values.state.path | quote }}
{{- end }}
{{- end -}}

{{/*
The URL infra-www proxies /api to.

Derived from the in-cluster api Service unless the caller overrides it. Fails
rather than guessing when the dashboard is enabled, the API is not, and no
external URL was given: the resulting deployment serves a dashboard whose every
data call 502s, which looks like a broken API rather than a misconfigured chart.
*/}}
{{- define "infra.apiURL" -}}
{{- $root := .root -}}
{{- $www := $root.Values.components.www -}}
{{- if $www.apiURL -}}
{{- $www.apiURL -}}
{{- else if $root.Values.components.api.enabled -}}
{{- printf "http://%s-api.%s.svc:%v" (include "infra.fullname" $root) $root.Release.Namespace $root.Values.components.api.service.port -}}
{{- else -}}
{{- fail "components.www is enabled but components.api is not, and components.www.apiURL is empty. The dashboard proxies /api to the control plane, so it needs an API to reach: enable components.api, set components.www.apiURL to an external control plane, or disable components.www." -}}
{{- end -}}
{{- end -}}

{{/*
Client credentials for infra-idp, from a secret whenever one is configured so
the tokens never appear in the rendered manifest.
*/}}
{{- define "infra.idpEnv" -}}
{{- $idp := .root.Values.components.idp -}}
{{- if $idp.existingSecret }}
- name: GOA_IDP_CLIENTS
  valueFrom:
    secretKeyRef:
      name: {{ $idp.existingSecret }}
      key: {{ $idp.existingSecretKey }}
{{- else }}
- name: GOA_IDP_CLIENTS
  value: {{ $idp.clients | quote }}
{{- end }}
{{- end -}}
