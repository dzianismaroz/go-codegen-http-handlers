package main

import "net/http"

const (
    validAuthToken = "100500"
    authHeader = "X-Auth"
)

func isAuthorized(r *http.Request) bool {
	return r.Header.Get(authHeader) == validAuthToken
}

// ------------------- HTTP handlers --------------------{{if .}} {{range $k , $v :=  .}}

func (h *{{$k}} ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path { {{range $i, $api := $v}}
    case "{{$api.Url}}":  {{ if $api.Auth }}
        if !isAuthorized(r) {
            http.Error(w, "unauthorized", 401)
            return
        } {{end}} {{ if ne $api.Method  "" }}
        if r.Method != "{{$api.Method}}" {
            http.Error(w, "not found", 404)
            return
        }{{end}}
        h.{{$api.Target.Name.Name}}(r.Context(), r)    {{end}}
    default:
        w.WriteHeader(http.StatusNotFound)
    }
}
{{end}}
{{end}}
