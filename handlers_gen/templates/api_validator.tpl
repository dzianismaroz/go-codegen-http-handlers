// ------------------- Validators --------------------

{{if .}}
{{range $i, $s := .}}
    {{$s.Name.Name}}
{{end}}

{{end}}

