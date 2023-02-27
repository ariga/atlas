{{- if eq .Env "dev" }}
    create table dev1 (c text);
{{- else  }}
    create table prod1 (c text);
{{- end }}