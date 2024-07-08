// ------------------- Validators -------------------- {{if .}}{{range $i, $s := .}}

    func (s *{{$s.Name.Name}}) extract(r *http.Request) {

    }

    func (s *{{$s.Name.Name}}) validate() error {
        return nil
    }{{end}}{{end}}




