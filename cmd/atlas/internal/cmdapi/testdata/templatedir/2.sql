{{- if eq .Env "dev" }}
    create table dev2 (c text);
    {{ template "shared/users" "dev2" }}
{{- else  }}
    create table prod2 (c text);
    {{ template "shared/users" "prod2" }}
{{- end }}