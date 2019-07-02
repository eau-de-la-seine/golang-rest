// Date 05/09/2018
// Author: Gokan EKINCI
// Godoc: https://godoc.org/net/http

package rest

import (
	"net/http"
	"errors"
	"reflect"
	"io/ioutil"
	"io"
	"encoding/json"
	"encoding/xml"
	"regexp"
	"fmt"
	"time"
	"strings"
	"github.com/eau-de-la-seine/golang-logger"
)

var log *logger.Logger = logger.NewConsoleLogger(logger.LEVEL_DEBUG)

// Code + Data
type HttpResponse interface {
	write(response http.ResponseWriter)
}

// HTTP RESPONSE (JSON/XML)
type ResponseWriter struct {
	customHeaders map[string]string
	contentType string
	statusCode int
	responseBody interface{}
	marshal func(interface{}) ([]byte, error)
}

func (r *ResponseWriter) write(response http.ResponseWriter) {
	response.WriteHeader(r.statusCode)
	response.Header().Set("Content-Type", r.contentType)

	for key, value := range r.customHeaders {
		response.Header().Set(key, value)
	}

	if marshallizedResponse, marshalErr := r.marshal(r.responseBody); marshalErr == nil {
		// Write HTTP response
		if _, err := response.Write(marshallizedResponse); err != nil {
			log.Debug("[ResponseWriter#write] response.Write => %s", err.Error())
		}
	} else {
		log.Debug("[ResponseWriter#write] marshal => %s", marshalErr.Error())
	}
}

// HTTP RESPONSE (FILE)
type FileResponseWriter struct {
	contentType string
	statusCode int
	contentLength int
	file io.Reader
	contentDisposition string
}

func (r *FileResponseWriter) write(response http.ResponseWriter) {
	if r.contentLength > 0 {
		response.Header().Set("Content-Length", string(r.contentLength))
	}

	response.Header().Set("Content-Disposition", r.contentDisposition)

	response.WriteHeader(r.statusCode)
	response.Header().Set("Content-Type", r.contentType)

	if _, copyErr := io.Copy(response, r.file); copyErr != nil {
		log.Debug("[FileResponseWriter#write] Copy => %s", copyErr.Error())
	}
}


// HTTP RESPONSE (NO-CONTENT)
type NoContentResponseWriter struct {}

func (r *NoContentResponseWriter) write(response http.ResponseWriter) {
	response.WriteHeader(http.StatusNoContent)
}

// HTTP RESPONSE (TEXT)

type TextResponseWriter struct {
	customHeaders map[string]string
	statusCode int
	responseBody string
}

func (r *TextResponseWriter) write(response http.ResponseWriter) {
	response.WriteHeader(r.statusCode)
	response.Header().Set("Content-Type", "text/plain")

	for key, value := range r.customHeaders {
		response.Header().Set(key, value)
	}

	if _, err := response.Write([]byte(r.responseBody)); err != nil {
		log.Debug("[TextResponseWriter#write] response.Write => %s", err.Error())
	}
}

// IMPLEMENTATIONS

func JsonResponse(statusCode int, responseBody interface{}, customHeaders map[string]string) HttpResponse {
	return &ResponseWriter{
		customHeaders: customHeaders,
		contentType: "application/json",
		statusCode: statusCode,
		responseBody: responseBody,
		marshal: json.Marshal}
}

func XmlResponse(statusCode int, responseBody interface{}, customHeaders map[string]string) HttpResponse {
	return &ResponseWriter{
		customHeaders: customHeaders,
		contentType: "application/xml",
		statusCode: statusCode,
		responseBody: responseBody,
		marshal: xml.Marshal}
}

type ErrorResponse struct {
	// time.Now().Format(time.RFC3339)
	Date string
	Message string
	Method string
	Path string
}

func JsonErrorResponse(statusCode int, request *http.Request, message string) HttpResponse {
	responseBody := &ErrorResponse{
		Date: time.Now().Format(time.RFC3339),
		Message: message,
		Method: request.Method,
		Path: request.URL.Path}

	return &ResponseWriter{
		contentType: "application/json",
		statusCode: statusCode,
		responseBody: responseBody,
		marshal: json.Marshal}
}

func XmlErrorResponse(statusCode int, request *http.Request, message string) HttpResponse {
	responseBody := &ErrorResponse{
		Date: time.Now().Format(time.RFC3339),
		Message: message,
		Method: request.Method,
		Path: request.URL.Path}

	return &ResponseWriter{
		contentType: "application/xml",
		statusCode: statusCode,
		responseBody: responseBody,
		marshal: xml.Marshal}
}

func FileResponse(statusCode int, contentType string, contentDisposition string, contentLength int, file io.Reader) HttpResponse {
	return &FileResponseWriter{
		contentType: contentType,
		contentDisposition: contentDisposition,
		contentLength: contentLength,
		statusCode: statusCode,
		file: file}
}

func NoContentResponse() HttpResponse {
	return &NoContentResponseWriter{}
}

func TextResponse(statusCode int, responseBody string) HttpResponse {
	return &TextResponseWriter{
		statusCode: statusCode,
		responseBody: responseBody}
}

type PathVariable struct {
	// Index of the pathVariable starting from zero. Ex: /{v0}/{v1}/path2/{v3}/path4
	pathIndex int

	// Variable name. Ex: v0, v1, v3
	variableName string
}

type Http struct {
	Response http.ResponseWriter
	Request *http.Request
	PathVariables map[string]string
	// TODO: For Golang 2, add generic `RequestBody T` here
}

type CustomHandler interface {
	GetRegexPath() *regexp.Regexp
	GetRequestBodyType() reflect.Type
	GetPathVariableNames() []PathVariable
	HasRequestBody() bool
	// TODO: For Golang 2, replace inputs type by `rest.Http`
	WriteHttpResponse(response http.ResponseWriter, inputs []reflect.Value)
}

type CustomHandlerImpl struct {
	regexPath *regexp.Regexp

	// Can be nil if no data
	pathVariableNames []PathVariable

	// Can be nil if handler has 1 input parameter, but must NOT be nil if handler has 2 input parameters
	requestBodyType reflect.Type

	// Deprecated: Will be replaced by generic function signature with Golang 2 generics
	// Signature of the handler must be:
	// 1. rest.Http (contains response, request, pathVariables)
	// 2. Object (HTTP Request Body generated by JSON or XML), optional
	// Return an `rest.HttpResponse`
	handlerValue reflect.Value
}

func NewCustomHandlerImpl(httpMethod string, path string, handlerFunction interface{}) CustomHandler {
	log.Debug("[NewCustomHandlerImpl] Method: '%s' | Path: '%s'", httpMethod, path)

	if handlerFunction == nil {
		panic("[NewCustomHandlerImpl] handler must not be `nil`")
	}

	assertValidPath(path)

	// Validate Handler
	handlerFunctionType := reflect.TypeOf(handlerFunction)
	validateHandler(httpMethod, handlerFunctionType)

	// Initialization
	obj := new(CustomHandlerImpl)
	obj.pathVariableNames = extractPathVariableNames(path)
	obj.regexPath = toRegexPath(path)
	
	// Type of param n°2 (Request body type)
	if handlerFunctionType.NumIn() == 2 {
		// Getting the underlying type of pointer-type (ex: *MyRequestBody => MyRequestBody)
		obj.requestBodyType = handlerFunctionType.In(1).Elem()
	}

	obj.handlerValue = reflect.ValueOf(handlerFunction)

	return obj
}

func (h *CustomHandlerImpl) GetRegexPath() *regexp.Regexp {
	return h.regexPath
}

func (h *CustomHandlerImpl) GetPathVariableNames() []PathVariable {
	return h.pathVariableNames
}

func (h *CustomHandlerImpl) HasRequestBody() bool {
	return h.requestBodyType != nil
}

func (h *CustomHandlerImpl) GetRequestBodyType() reflect.Type {
	return h.requestBodyType
}

func (h *CustomHandlerImpl) WriteHttpResponse(response http.ResponseWriter, inputs []reflect.Value) {
	if impl, ok := h.handlerValue.Call(inputs)[0].Interface().(HttpResponse); ok {
		impl.write(response)
	}
}

// Map of HttpMethod/CustomHandlers
type Routes map[string][]CustomHandler

func NewRoutes() Routes {
	return make(Routes, 0)
}

// Notes:
// * Don't need to check httpMethod
// * path, handler will be checked in `NewCustomHandlerImpl()`
func (routes Routes) addRoute(httpMethod string, path string, handler interface{}) Routes {
	if _, exists := routes[httpMethod]; !exists {
		routes[httpMethod] = make([]CustomHandler, 0)
	}

	routes[httpMethod] = append(
		routes[httpMethod],
		NewCustomHandlerImpl(httpMethod, path, handler))

	return routes
}

func (routes Routes) GET(path string, handler interface{}) Routes {
	return routes.addRoute(http.MethodGet, path, handler)
}

func (routes Routes) POST(path string, handler interface{}) Routes {
	return routes.addRoute(http.MethodPost, path, handler)
}

func (routes Routes) PUT(path string, handler interface{}) Routes {
	return routes.addRoute(http.MethodPut, path, handler)
}

func (routes Routes) PATCH(path string, handler interface{}) Routes {
	return routes.addRoute(http.MethodPatch, path, handler)
}

func (routes Routes) DELETE(path string, handler interface{}) Routes {
	return routes.addRoute(http.MethodDelete, path, handler)
}

type FilterFunc func(http.ResponseWriter, *http.Request) bool
type filterMap map[string][]FilterFunc
type Filters struct {
	filters filterMap
}

func NewFilters() *Filters {
	obj := new(Filters)
	obj.filters = make(filterMap, 0)
	obj.filters["pre"] = make([]FilterFunc, 0)
	obj.filters["post"] = make([]FilterFunc, 0)
	return obj
}

func (filters *Filters) AddPreFilter(filter FilterFunc) *Filters {
	if filter == nil {
		panic("[Filters#AddPreFilter] 'filter' must not be `nil`")
	}

	filters.filters["pre"] = append(filters.filters["pre"], filter)
	return filters
}

func (filters *Filters) AddPostFilter(filter FilterFunc) *Filters {
	if filter == nil {
		panic("[Filters#AddPreFilter] 'filter' must not be `nil`")
	}

	filters.filters["post"] = append(filters.filters["post"], filter)
	return filters
}

func unmarshal(contentType string, rawData []byte, objectToFill interface{}) error {
	switch contentType {
		case "application/xml":
			return xml.Unmarshal(rawData, objectToFill)
		default:
			return json.Unmarshal(rawData, objectToFill)
	}
}

func isHttpMethodBodyable(httpMethod string) bool {
    switch httpMethod {
		case
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete:
			return true
	}

	return false
}

// Deprecated: Will be removed with Golang 2's generics
func validateHandler(httpMethod string, handlerFunctionType reflect.Type) {
	log.Debug("[validateHandler] httpMethod => '%s' | isHttpMethodBodyable => '%t' | handlerFunctionType => %s",
		httpMethod,
		isHttpMethodBodyable(httpMethod),
		handlerFunctionType)

	if handlerFunctionType.Kind() != reflect.Func {
		panic("[validateHandler] Parameter 'handlerFunctionType' is not a `func`")
	}
	log.Debug("[validateHandler] NumIn => %d | NumOut => %d", handlerFunctionType.NumIn(), handlerFunctionType.NumOut())

	numIn := handlerFunctionType.NumIn()
	if !(numIn == 1 || numIn == 2) {
		panic(fmt.Sprintf("[validateHandler] Parameter 'handlerFunctionType' must have 1 or 2 input parameters but had %d parameters", numIn))
	} else if isHttpMethodBodyable(httpMethod) && numIn == 1 {
		panic(fmt.Sprintf("[validateHandler] Parameter 'handlerFunctionType' for '%s' HTTP method must have 2 parameters", httpMethod))
	} else if !isHttpMethodBodyable(httpMethod) && numIn == 2 {
		panic(fmt.Sprintf("[validateHandler] Parameter 'handlerFunctionType' for '%s' HTTP method must have 1 parameters", httpMethod))
	}

	firstParameterType := handlerFunctionType.In(0)
	httpType := reflect.TypeOf((**Http)(nil)).Elem()
	if firstParameterType != httpType {
		panic(fmt.Sprintf("[validateHandler] Parameter 'handlerFunctionType' parameter n°1 type must be 'rest.Http' but was '%s'", firstParameterType))
	}

	if numIn == 2 {
		secondParameterType := handlerFunctionType.In(1)
		if secondParameterType.Kind() != reflect.Ptr {
			panic(fmt.Sprintf("[validateHandler] Parameter 'handlerFunctionType' parameter n°2 type must be a pointer like '*%s' but was '%s'", secondParameterType, secondParameterType))
		}
	}

	numOut := handlerFunctionType.NumOut()
	if numOut != 1 {
		panic(fmt.Sprintf("[validateHandler] Parameter 'handlerFunctionType' must have 1 output parameter but had %d parameter(s)", numOut))
	}
	returnType := handlerFunctionType.Out(0)
	httpResponseType := reflect.TypeOf((*HttpResponse)(nil)).Elem()
	if returnType != httpResponseType {
		panic(fmt.Sprintf("[validateHandler] Parameter 'handlerFunctionType' return type must be 'rest.HttpResponse' but was '%s'", returnType))
	}
}

// Deprecated: Will be removed with Golang 2's generics
func inputsWithoutRequestBody(http *Http) []reflect.Value {
	return []reflect.Value{reflect.ValueOf(http)}
}

// Deprecated: Will be removed with Golang 2's generics
func inputsWithRequestBody(http *Http, requestBody interface{}) []reflect.Value {
	// log.Debug("[inputsWithRequestBody] RequestBody type and value => [%+v][%+v]", reflect.TypeOf(requestBody), requestBody)
	return []reflect.Value{ reflect.ValueOf(http), reflect.ValueOf(requestBody) }
}

// Valid paths:
// /
// /path1
// /path1/pa-th-2/3
// /path1/{pa-th-2}/3
func isValidPath(path string) (bool, error) {
	if path == "/" {
		return true, nil
	}

	subPathPattern := `[a-z0-9]+(-?[a-z0-9]+)*`
	pathPattern := fmt.Sprintf(`^(/(({%s})|(%s)))+$`, subPathPattern, subPathPattern)
	return regexp.MatchString(pathPattern, path)
}

func assertValidPath(path string) {
	if ok, err := isValidPath(path); !ok {
		panic(fmt.Sprintf("[assertValidPath] Path '%s' didn't matched regex pattern -> '%s'", path, err))
	}
}

func removeBraces(key string) string {
	return strings.Replace(
		strings.Replace(key, "}", "", 1),
		"{", "", 1)
}

func extractPathVariableNames(path string) []PathVariable {
	extractedPathVariableNames := make([]PathVariable, 0)
	separator := "/"
	prefix := "{"

	pathParts := strings.Split(path, separator)
	for partIndex, partValue := range pathParts {
		if strings.HasPrefix(partValue, prefix) {
			extractedPathVariableNames = append(
				extractedPathVariableNames,
				PathVariable{pathIndex: partIndex - 1, variableName: removeBraces(partValue)})
		}
	}

	if len(extractedPathVariableNames) == 0 {
		return nil
	}

	return extractedPathVariableNames
}

// This function may need to be optimized, because it will be executed each time a request is received
func extractPathVariableValues(path string, pathVariables []PathVariable) map[string]string {
	extractedPathVariableValues := make(map[string]string, 0)
	separator := "/"
	pathParts := strings.Split(path, separator)

	for _, pathVariable := range pathVariables {
		extractedPathVariableValues[pathVariable.variableName] = pathParts[pathVariable.pathIndex + 1]
	}

	return extractedPathVariableValues
}

func toRegexPath(path string) *regexp.Regexp {
	regexPart := "[a-zA-Z0-9_-]+"
	regexPathVariableName := regexp.MustCompile("\\{(.+?)\\}")
	return regexp.MustCompile(regexPathVariableName.ReplaceAllString(path, regexPart))
}

func toRequestBodyObject(request *http.Request, requestBodyType reflect.Type) (interface{}, error) {
	bodyBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	log.Debug("[toRequestBodyObject] bodyBytes => %s", bodyBytes)

	objectToFill := reflect.New(requestBodyType).Interface()
	if unmarshalErr := unmarshal(request.Header.Get("Content-Type"), bodyBytes, objectToFill); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return objectToFill, nil
}

// type Handler interface {
//    ServeHTTP(ResponseWriter, *Request)
// }
type Dispatcher struct {
	routes Routes
	preFilters []FilterFunc
	postFilters []FilterFunc
}

func NewDispatcher(routes Routes, filters *Filters) *Dispatcher {
	if routes == nil {
		panic("[NewDispatcher] routes must not be `nil`")
	}

	dispatcher := new(Dispatcher)
	dispatcher.routes = routes

	if filters == nil {
		return dispatcher
	}

	if len(filters.filters["pre"]) > 0 {
		dispatcher.preFilters = filters.filters["pre"]
	}

	if len(filters.filters["post"]) > 0 {
		dispatcher.postFilters = filters.filters["post"]
	}

	return dispatcher
}

func (dispatcher *Dispatcher) getHandler(httpMethod string, calledPath string) (CustomHandler, error) {
	for _, handler := range dispatcher.routes[httpMethod] {
		if handler.GetRegexPath().MatchString(calledPath) {
			return handler, nil
		}
	}

	// Error = 404 not found, otherwise a 200 response will be returned by default
	return nil, errors.New(fmt.Sprintf("[Dispatcher#getHandler] Route does NOT exists => Method: '%s' | Path: '%s'", httpMethod, calledPath))
}

func executeFilters(response http.ResponseWriter, request *http.Request, filters []FilterFunc) bool {
	for _, filter := range filters {
		if !filter(response, request) {
			return false
		}
	}

	return true
}

func (dispatcher *Dispatcher) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	calledPath := request.URL.Path
	handler, err := dispatcher.getHandler(request.Method, calledPath)
	if err != nil {
		// Printing debug
		log.Debug(err.Error())
		response.WriteHeader(404)
		return
	}

	log.Debug("[Dispatcher#ServeHTTP] => Method: '%s' | Path: '%s'", request.Method, calledPath)

	// Executing pre-filters
	if !executeFilters(response, request, dispatcher.preFilters) {
		return
	}

	// Executing handler
	pathVariableValues := extractPathVariableValues(calledPath, handler.GetPathVariableNames())
	http := &Http{Response: response, Request: request, PathVariables: pathVariableValues}
	if !handler.HasRequestBody() {
		inputs := inputsWithoutRequestBody(http)
		handler.WriteHttpResponse(response, inputs)
	} else {
		if requestBody, err := toRequestBodyObject(request, handler.GetRequestBodyType()); err != nil {
			log.Debug("[Dispatcher#ServeHTTP][toRequestBodyObject] %s", err.Error())
			return
		} else {
			inputs := inputsWithRequestBody(http, requestBody)
			handler.WriteHttpResponse(response, inputs)
		}
	}

	// Executing post-filters
	executeFilters(response, request, dispatcher.postFilters)
}