package main

import (
	"net/http"
	"context"
	"errors"
	"slices"
	"strconv"
	"encoding/json"
	"io"
	"net/url"
)

func unpackValues(valuesMap map[string][]string) map[string]string {
	var unpackedMap = make(map[string]string)
	for key, value := range valuesMap {
		if len(value) > 0 {
			unpackedMap[key] = value[0]
		}
	}
	return unpackedMap
}
type ResError struct { Error string `json:"error"` }


	
	 /* POST */
		func (h *MyApi) wrapperCreate(w http.ResponseWriter, r *http.Request) (interface{}, *ApiError) {
			var rawParams map[string]string
			
			if (r.Method != "POST") {
					return nil, &ApiError{http.StatusNotAcceptable, errors.New("bad method")}
			}
			
			
			
				var authValue string
				auth := r.Header.Get("X-Auth")
				if len(auth) > 0 {
					authValue = string(r.Header.Get("X-Auth"))
				}
			if authValue != "100500" {
				return nil, &ApiError{http.StatusForbidden, errors.New("unauthorized")}
			}
			

			if (r.Method == "POST") {
				body, err := io.ReadAll(r.Body)
				if (err != nil) {
					return nil, &ApiError{http.StatusInternalServerError, err}
					}
					parsedBody, err := url.ParseQuery(string(body))
					if (err != nil) {
						return nil, &ApiError{http.StatusInternalServerError, err}
						}
						rawParams = unpackValues(parsedBody)
						} else {
							 query := r.URL.Query()
							rawParams = unpackValues(query)
							}
							/* Validate Params */
							var validatedParams CreateParams
							var rawValue string
							var ok bool
							
							var AgeParam int
							
							rawValue, ok = rawParams["age"]
							
							
							

							if ok {
								
								var err error
								AgeParam, err = strconv.Atoi(rawValue)
								if (err != nil) {
									return nil, &ApiError{http.StatusBadRequest, errors.New("age must be int")}
								}
								
								}

								/* Default */
								
								
								
								/* Min */
								AgeMin := 0
								
									if AgeParam < AgeMin {
									return nil, &ApiError{http.StatusBadRequest, errors.New("age must be >= 0")}
								}
								
								
								
								/* Max */
								AgeMax := 128
								
								if AgeParam > AgeMax {
									return nil, &ApiError{http.StatusBadRequest, errors.New("age must be <= 128")}
								}
								
								
														 
								
								validatedParams.Age = AgeParam
							
							var LoginParam string
							
							rawValue, ok = rawParams["login"]
							
							
							
							/* Required */
							if !ok {
							return nil, &ApiError{http.StatusBadRequest, errors.New("login must be not empty")}
							}
							

							if ok {
								
								LoginParam = rawValue
								
								}

								/* Default */
								
								
								
								/* Min */
								LoginMin := 10
								
								if len(LoginParam) < LoginMin {
									return nil, &ApiError{http.StatusBadRequest, errors.New("login len must be >= 10")}
									}
									
								
								
														 
								
								validatedParams.Login = LoginParam
							
							var NameParam string
							
							/* full_name */
							rawValue, ok = rawParams["full_name"]
							
							
							

							if ok {
								
								NameParam = rawValue
								
								}

								/* Default */
								
								
								
								
														 
								
								validatedParams.Name = NameParam
							
							var StatusParam string
							
							rawValue, ok = rawParams["status"]
							
							
							

							if ok {
								
								StatusParam = rawValue
								
								}

								/* Default */
								
								
								if StatusParam == "" {
									StatusParam = "user"
								}
								
								
								
								
								
								
														 
							/* Enum */
							 
							StatusOptions := []string {"user","moderator","admin"}
							if !slices.Contains(StatusOptions, StatusParam) {
								return nil, &ApiError{http.StatusBadRequest, errors.New("status must be one of [user, moderator, admin]")}
							}
							
								
								validatedParams.Status = StatusParam
							
			ctx := context.Background()
			res, err := h.Create(ctx, validatedParams)

			if err != nil {
				if apiErr, ok := err.(ApiError); ok {
					return nil, &ApiError{apiErr.HTTPStatus, err}
				}
					return nil, &ApiError{http.StatusInternalServerError, err}
			}
			return res, nil
		}
	
	 /*  */
		func (h *MyApi) wrapperProfile(w http.ResponseWriter, r *http.Request) (interface{}, *ApiError) {
			var rawParams map[string]string
			
			
			

			if (r.Method == "POST") {
				body, err := io.ReadAll(r.Body)
				if (err != nil) {
					return nil, &ApiError{http.StatusInternalServerError, err}
					}
					parsedBody, err := url.ParseQuery(string(body))
					if (err != nil) {
						return nil, &ApiError{http.StatusInternalServerError, err}
						}
						rawParams = unpackValues(parsedBody)
						} else {
							 query := r.URL.Query()
							rawParams = unpackValues(query)
							}
							/* Validate Params */
							var validatedParams ProfileParams
							var rawValue string
							var ok bool
							
							var LoginParam string
							
							rawValue, ok = rawParams["login"]
							
							
							
							/* Required */
							if !ok {
							return nil, &ApiError{http.StatusBadRequest, errors.New("login must be not empty")}
							}
							

							if ok {
								
								LoginParam = rawValue
								
								}

								/* Default */
								
								
								
								
														 
								
								validatedParams.Login = LoginParam
							
			ctx := context.Background()
			res, err := h.Profile(ctx, validatedParams)

			if err != nil {
				if apiErr, ok := err.(ApiError); ok {
					return nil, &ApiError{apiErr.HTTPStatus, err}
				}
					return nil, &ApiError{http.StatusInternalServerError, err}
			}
			return res, nil
		}
	

	
	 /* POST */
		func (h *OtherApi) wrapperCreate(w http.ResponseWriter, r *http.Request) (interface{}, *ApiError) {
			var rawParams map[string]string
			
			if (r.Method != "POST") {
					return nil, &ApiError{http.StatusNotAcceptable, errors.New("bad method")}
			}
			
			
			
				var authValue string
				auth := r.Header.Get("X-Auth")
				if len(auth) > 0 {
					authValue = string(r.Header.Get("X-Auth"))
				}
			if authValue != "100500" {
				return nil, &ApiError{http.StatusForbidden, errors.New("unauthorized")}
			}
			

			if (r.Method == "POST") {
				body, err := io.ReadAll(r.Body)
				if (err != nil) {
					return nil, &ApiError{http.StatusInternalServerError, err}
					}
					parsedBody, err := url.ParseQuery(string(body))
					if (err != nil) {
						return nil, &ApiError{http.StatusInternalServerError, err}
						}
						rawParams = unpackValues(parsedBody)
						} else {
							 query := r.URL.Query()
							rawParams = unpackValues(query)
							}
							/* Validate Params */
							var validatedParams OtherCreateParams
							var rawValue string
							var ok bool
							
							var ClassParam string
							
							rawValue, ok = rawParams["class"]
							
							
							

							if ok {
								
								ClassParam = rawValue
								
								}

								/* Default */
								
								
								if ClassParam == "" {
									ClassParam = "warrior"
								}
								
								
								
								
								
								
														 
							/* Enum */
							 
							ClassOptions := []string {"warrior","sorcerer","rouge"}
							if !slices.Contains(ClassOptions, ClassParam) {
								return nil, &ApiError{http.StatusBadRequest, errors.New("class must be one of [warrior, sorcerer, rouge]")}
							}
							
								
								validatedParams.Class = ClassParam
							
							var LevelParam int
							
							rawValue, ok = rawParams["level"]
							
							
							

							if ok {
								
								var err error
								LevelParam, err = strconv.Atoi(rawValue)
								if (err != nil) {
									return nil, &ApiError{http.StatusBadRequest, errors.New("level must be int")}
								}
								
								}

								/* Default */
								
								
								
								/* Min */
								LevelMin := 1
								
									if LevelParam < LevelMin {
									return nil, &ApiError{http.StatusBadRequest, errors.New("level must be >= 1")}
								}
								
								
								
								/* Max */
								LevelMax := 50
								
								if LevelParam > LevelMax {
									return nil, &ApiError{http.StatusBadRequest, errors.New("level must be <= 50")}
								}
								
								
														 
								
								validatedParams.Level = LevelParam
							
							var NameParam string
							
							/* account_name */
							rawValue, ok = rawParams["account_name"]
							
							
							

							if ok {
								
								NameParam = rawValue
								
								}

								/* Default */
								
								
								
								
														 
								
								validatedParams.Name = NameParam
							
							var UsernameParam string
							
							rawValue, ok = rawParams["username"]
							
							
							
							/* Required */
							if !ok {
							return nil, &ApiError{http.StatusBadRequest, errors.New("username must be not empty")}
							}
							

							if ok {
								
								UsernameParam = rawValue
								
								}

								/* Default */
								
								
								
								/* Min */
								UsernameMin := 3
								
								if len(UsernameParam) < UsernameMin {
									return nil, &ApiError{http.StatusBadRequest, errors.New("username len must be >= 3")}
									}
									
								
								
														 
								
								validatedParams.Username = UsernameParam
							
			ctx := context.Background()
			res, err := h.Create(ctx, validatedParams)

			if err != nil {
				if apiErr, ok := err.(ApiError); ok {
					return nil, &ApiError{apiErr.HTTPStatus, err}
				}
					return nil, &ApiError{http.StatusInternalServerError, err}
			}
			return res, nil
		}
	



func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
var response interface{}
var error *ApiError
	switch r.URL.Path {
		
		case "/user/create":
			response, error = srv.wrapperCreate(w,r)
		
		case "/user/profile":
			response, error = srv.wrapperProfile(w,r)
		
		default:
			error = &ApiError{http.StatusNotFound, errors.New("unknown method")}
		}
			
			res := make(map[string]interface{})
			if error != nil {
			w.WriteHeader(error.HTTPStatus)
			res["error"] = error.Error()
		} else {
			res["error"] = ""
			res["response"] = response
		}
		jsonRes, _ := json.Marshal(res)
		w.Write(jsonRes)
		}
		
func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
var response interface{}
var error *ApiError
	switch r.URL.Path {
		
		case "/user/create":
			response, error = srv.wrapperCreate(w,r)
		
		default:
			error = &ApiError{http.StatusNotFound, errors.New("unknown method")}
		}
			
			res := make(map[string]interface{})
			if error != nil {
			w.WriteHeader(error.HTTPStatus)
			res["error"] = error.Error()
		} else {
			res["error"] = ""
			res["response"] = response
		}
		jsonRes, _ := json.Marshal(res)
		w.Write(jsonRes)
		}
		
