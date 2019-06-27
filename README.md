# Golang REST server-side framework

A simple REST framework for creating server-side web application with Golang 1

Status: Experimental

TODO: return 404 if route not found


## Types

* `Routes`: Create HTTP GET, POST, PUT, PATCH, DELETE routes. Check **Handler Signature** for more informations.

```
routes := rest.NewRoutes().
			GET(PATH, getHandler).
			POST(PATH, postHandler)
```

* `Filters`: Create Pre and Post-filters, executed before and after your handler. Your filters must return `true` if everything is OK, or `false` for stopping the treatment.

```
filters := rest.NewFilters().
	AddPreFilter(func(response http.ResponseWriter, request *http.Request) bool {
		fmt.Println("Filter 1")
		return true
	}).
	AddPostFilter(func(response http.ResponseWriter, request *http.Request) bool {
		fmt.Println("Filter 2")
		return true
	})
```


* `Dispatcher`: Initialized with your routes and filters, it implements Golang's `ServeHTTP()` function.

```
dispatcher := rest.NewDispatcher(routes, filters)

const PORT = ":8080"

s := &http.Server{
	Addr:           PORT,
	Handler:        dispatcher,
	ReadTimeout:    10 * time.Second,
	WriteTimeout:   10 * time.Second,
	MaxHeaderBytes: 1 << 20,
}
```



## Handler Signature

```
// If you don't handle a HTTP Request
func(http *rest.Http) rest.HttpResponse

// If you handle a HTTP Request
func(http *rest.Http, requestBody *YourType) rest.HttpResponse
```

The `rest.Http` structure contains the following fields:
* `Response`: Golang's `http.ResponseWriter` type
* `Request`: Golang's `http.Request` type
* `PathVariables`: A map containing a pair of key from the given path (when you create your route with placeholders => `/path/{your-key}`), and its value
* Work In Progress for Golang 2: `RequestBody`



## `HttpResponse` implementation functions

### Returning JSON or XML response

* `JsonResponse(statusCode int, responseBody interface{})`
* `XmlResponse(statusCode int, responseBody interface{})`


### Returning JSON or XML formatted error reponse

* `JsonErrorResponse(statusCode int, request *http.Request, message string)`
* `XmlErrorResponse(statusCode int, request *http.Request, message string)`


### Returning file

* `FileResponse(statusCode int, contentType string, contentDisposition string, contentLength int, file io.Reader)`


### Other cases

* `TextResponse(statusCode int, responseBody string)`
* `NoContentResponse()`



## Example of use (Golang 1)

```
package main

import (
	"github.com/eau-de-la-seine/golang-rest"
	"net/http"
	"time"
	"fmt"
)

// Lower case fields are not visible by Golang reflection
type DemoBody struct {
	A int `json:"a"`
	B string `json:"b"`
}

func main() {
	getHandler := func(http *rest.Http) rest.HttpResponse {
		return rest.TextResponse(200, `{ "text": "Lambda hello" }`)
	}

	postHandler := func(http *rest.Http, body *DemoBody) rest.HttpResponse {
		return rest.JsonResponse(200, &DemoBody{A: body.A, B: body.B})
	}

	const PATH = "/path/path"

	routes := rest.NewRoutes().
		GET(PATH, getHandler).
		POST(PATH, postHandler)

	filters := rest.NewFilters().
		AddPreFilter(func(response http.ResponseWriter, request *http.Request) bool {
			fmt.Println("Filter 1")
			return true
		}).
		AddPostFilter(func(response http.ResponseWriter, request *http.Request) bool {
			fmt.Println("Filter 2")
			return true
		})

	dispatcher := rest.NewDispatcher(routes, filters)

	const PORT = ":8080"

	s := &http.Server{
		Addr:           PORT,
		Handler:        dispatcher,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	fmt.Println(s.ListenAndServe())
}
```
