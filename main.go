package main

// this is the program for which your code generator will write code
// run via go test -v as usual

// this code is commented out so that it does not appear in the test coverage

import (
	"fmt"
	"net/http"
)

func main() {
	// the ServeHTTP method will be called on the MyApi structure
	http.Handle("/user/", NewMyApi())

	fmt.Println("starting server at :8080")
	http.ListenAndServe(":8080", nil)
}
