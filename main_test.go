package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func CheckoutDummy(w http.ResponseWriter, r *http.Request) {

}

var (
	client = &http.Client{Timeout: time.Second}
)

type Case struct {
	Method string // GET by default in http.NewRequest if an empty string is passed
	Path   string
	Query  string
	Auth   bool
	Status int
	Result interface{}
}

const (
	ApiUserCreate  = "/user/create"
	ApiUserProfile = "/user/profile"
)

// CaseResponse
type CR map[string]interface{}

func TestMyApi(t *testing.T) {
	ts := httptest.NewServer(NewMyApi())

	cases := []Case{
		Case{ // successful request
			Path:   ApiUserProfile,
			Query:  "login=rvasily",
			Status: http.StatusOK,
			Result: CR{
				"error": "",
				"response": CR{
					"id":        42,
					"login":     "rvasily",
					"full_name": "Vasily Romanov",
					"status":    20,
				},
			},
		},
		Case{ // successful request - POST
			Path:   ApiUserProfile,
			Method: http.MethodPost,
			Query:  "login=rvasily",
			Status: http.StatusOK,
			Result: CR{
				"error": "",
				"response": CR{
					"id":        42,
					"login":     "rvasily",
					"full_name": "Vasily Romanov",
					"status":    20,
				},
			},
		},
		Case{ // validation worked - login must not be empty
			Path:   ApiUserProfile,
			Query:  "",
			Status: http.StatusBadRequest,
			Result: CR{
				"error": "login must me not empty",
			},
		},
		Case{ // received a general purpose error - your code itself substituted 500
			Path:   ApiUserProfile,
			Query:  "login=bad_user",
			Status: http.StatusInternalServerError,
			Result: CR{
				"error": "bad user",
			},
		},
		Case{ // got a specialized error - your code set the 404 status from there
			Path:   ApiUserProfile,
			Query:  "login=not_exist_user",
			Status: http.StatusNotFound,
			Result: CR{
				"error": "user not exist",
			},
		},
		// ------
		Case{ // this is what your ServeHTTP should respond to - if it receives something unknown (for example, when it processes /user/)
			Path:   "/user/unknown",
			Query:  "login=not_exist_user",
			Status: http.StatusNotFound,
			Result: CR{
				"error": "unknown method",
			},
		},
		// ------
		Case{ // create a user
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=mr.moderator&age=32&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusOK,
			Auth:   true,
			Result: CR{
				"error": "",
				"response": CR{
					"id": 43,
				},
			},
		},
		Case{ // the user has really been created
			Path:   ApiUserProfile,
			Query:  "login=mr.moderator",
			Status: http.StatusOK,
			Result: CR{
				"error": "",
				"response": CR{
					"id":        43,
					"login":     "mr.moderator",
					"full_name": "Ivan_Ivanov",
					"status":    10,
				},
			},
		},

		Case{ // POST only
			Path:   ApiUserCreate,
			Method: http.MethodGet,
			Query:  "login=mr.moderator&age=32&status=moderator&full_name=GetMethod",
			Status: http.StatusNotAcceptable,
			Auth:   true,
			Result: CR{
				"error": "bad method",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "any_params=123",
			Status: http.StatusForbidden,
			Auth:   false,
			Result: CR{
				"error": "unauthorized",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=mr.moderator&age=32&status=moderator&full_name=New_Ivan",
			Status: http.StatusConflict,
			Auth:   true,
			Result: CR{
				"error": "user mr.moderator exist",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "&age=32&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "login must me not empty",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=new_m&age=32&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "login len must be >= 10",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=new_moderator&age=ten&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "age must be int",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=new_moderator&age=-1&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "age must be >= 0",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=new_moderator&age=256&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "age must be <= 128",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=new_moderator&age=32&status=adm&full_name=Ivan_Ivanov",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "status must be one of [user, moderator, admin]",
			},
		},
		Case{ // status by default
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=new_moderator3&age=32&full_name=Ivan_Ivanov",
			Status: http.StatusOK,
			Auth:   true,
			Result: CR{
				"error": "",
				"response": CR{
					"id": 44,
				},
			},
		},
		Case{ // handle unknown error
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=bad_username&age=32&full_name=Ivan_Ivanov",
			Status: http.StatusInternalServerError,
			Auth:   true,
			Result: CR{
				"error": "bad user",
			},
		},
	}

	runTests(t, ts, cases)
}

func TestOtherApi(t *testing.T) {
	ts := httptest.NewServer(NewOtherApi())

	cases := []Case{
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "username=I3apBap&level=1&class=barbarian&account_name=Vasily",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "class must be one of [warrior, sorcerer, rouge]",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "username=I3apBap&level=1&class=warrior&account_name=Vasily",
			Status: http.StatusOK,
			Auth:   true,
			Result: CR{
				"error": "",
				"response": CR{
					"id":        12,
					"login":     "I3apBap",
					"full_name": "Vasily",
					"level":     1,
				},
			},
		},
	}

	runTests(t, ts, cases)
}

func runTests(t *testing.T, ts *httptest.Server, cases []Case) {
	for idx, item := range cases {
		var (
			err      error
			result   interface{}
			expected interface{}
			req      *http.Request
		)

		caseName := fmt.Sprintf("case %d: [%s] %s %s", idx, item.Method, item.Path, item.Query)

		if item.Method == http.MethodPost {
			reqBody := strings.NewReader(item.Query)
			req, err = http.NewRequest(item.Method, ts.URL+item.Path, reqBody)
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		} else {
			req, err = http.NewRequest(item.Method, ts.URL+item.Path+"?"+item.Query, nil)
		}

		if item.Auth {
			req.Header.Add("X-Auth", "100500")
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("[%s] request error: %v", caseName, err)
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		// fmt.Printf("[%s] body: %s\n", caseName, string(body))

		if resp.StatusCode != item.Status {
			t.Errorf("[%s] expected http status %v, got %v", caseName, item.Status, resp.StatusCode)
			continue
		}

		err = json.Unmarshal(body, &result)
		if err != nil {
			t.Errorf("[%s] cant unpack json: %v", caseName, err)
			continue
		}

		// reflect.DeepEqual does not work if we receive different types
		// and there they come with different types (string VS interface{}) compared to what is in the expected result
		// this dirty little hack converts data first to json and then back to interface - getting compatible results
		// do not use this in production code - you must explicitly write that the interface is expected or use another approach with the exact response format
		data, err := json.Marshal(item.Result)
		json.Unmarshal(data, &expected)

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("[%d] results not match\nGot: %#v\nExpected: %#v", idx, result, item.Result)
			continue
		}
	}
}
