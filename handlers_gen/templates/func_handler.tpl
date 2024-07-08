
func (h *{{.receiver}} ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case "{{.URL}}":
        h.{{.FuncName}}(r.Context(), r)
    default:
        w.WriteHeader(http.StatusNotFound)
    }
}
