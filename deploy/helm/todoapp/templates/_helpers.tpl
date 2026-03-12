{{- define "todoapp.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "todoapp.fullname" -}}
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

{{- define "todoapp.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "todoapp.labels" -}}
helm.sh/chart: {{ include "todoapp.chart" . }}
app.kubernetes.io/name: {{ include "todoapp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "todoapp.selectorLabels" -}}
app.kubernetes.io/name: {{ include "todoapp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "todoapp.commonEnvConfigMapName" -}}
{{- printf "%s-common-env" (include "todoapp.fullname" .) -}}
{{- end -}}

{{- define "todoapp.postgresServiceName" -}}
{{- printf "%s-postgres" (include "todoapp.fullname" .) -}}
{{- end -}}

{{- define "todoapp.vaultServiceName" -}}
{{- printf "%s-vault" (include "todoapp.fullname" .) -}}
{{- end -}}

{{- define "todoapp.pubsubServiceName" -}}
{{- printf "%s-pubsub-emulator" (include "todoapp.fullname" .) -}}
{{- end -}}

{{- define "todoapp.mcpServiceName" -}}
{{- printf "%s-mcp-gateway" (include "todoapp.fullname" .) -}}
{{- end -}}

{{- define "todoapp.secretName" -}}
{{- if .Values.env.secrets.existingSecret -}}
{{- .Values.env.secrets.existingSecret -}}
{{- else -}}
{{- printf "%s-secrets" (include "todoapp.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "todoapp.pubsubProjectSpec" -}}
{{- printf "%s,%s:%s,%s:%s:%s,%s" .Values.pubsub.projectId .Values.pubsub.topicIds.todo .Values.pubsub.subscriptionIds.todoEvents .Values.pubsub.topicIds.chatMessages .Values.pubsub.subscriptionIds.chatEvents .Values.pubsub.subscriptionIds.chatTitleEvents .Values.pubsub.topicIds.actionApprovals -}}
{{- end -}}
