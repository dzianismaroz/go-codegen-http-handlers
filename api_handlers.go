package main

import "net/http"

const (
    validAuthToken = "100500"
    authHeader = "X-Auth"
)

func isAuthorized(r *http.Request) bool {
	return r.Header.Get(authHeader) == validAuthToken
}

// ------------------- HTTP handlers -------------------- 

func (h *MyApi ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path { 
    case "/user/profile":   
        h.Profile(r.Context(), r)    
    case "/user/create":  
        if !isAuthorized(r) {
            http.Error(w, "unauthorized", 401)
            return
        }  
        if r.Method != "POST" {
            http.Error(w, "not found", 404)
            return
        }
        h.Create(r.Context(), r)    
    default:
        w.WriteHeader(http.StatusNotFound)
    }
}


func (h *OtherApi ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path { 
    case "/user/create":  
        if !isAuthorized(r) {
            http.Error(w, "unauthorized", 401)
            return
        }  
        if r.Method != "POST" {
            http.Error(w, "not found", 404)
            return
        }
        h.Create(r.Context(), r)    
    default:
        w.WriteHeader(http.StatusNotFound)
    }
}


// ------------------- Validators -------------------- 

    func (s *ProfileParams) extract(r *http.Request) {

    }

    func (s *ProfileParams) validate() error {
        return nil
    }

    func (s *CreateParams) extract(r *http.Request) {

    }

    func (s *CreateParams) validate() error {
        return nil
    }

    func (s *OtherCreateParams) extract(r *http.Request) {

    }

    func (s *OtherCreateParams) validate() error {
        return nil
    }




