package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type Data interface{}

type Response struct {
	Error    string `json:"error"`
	Response Data   `json:"response,omitempty"`
}

func (in *ProfileParams) Unpack(query string) error {
	values, err := url.ParseQuery(query)
	if err != nil {
		return err
	}

	login := values.Get("login")
	if len(login) == 0 {
		return errors.New("login must be not empty")
	}

	in.Login = login
	return nil
}

func (in *CreateParams) Unpack(query string) error {
	values, err := url.ParseQuery(query)
	if err != nil {
		return err
	}

	login := values.Get("login")
	if len(login) == 0 {
		return errors.New("login must be not empty")
	}

	if len(login) < 10 {
		return errors.New("login len must be >= 10")
	}
	in.Login = login
	name := values.Get("full_name")
	in.Name = name
	status := values.Get("status")
	if status == "" {
		status = "user"
	}

	if status != "user" && status != "moderator" && status != "admin" {
		return errors.New("status must be one of [user, moderator, admin]")
	}
	in.Status = status
	ageRaw := values.Get("age")
	age, err := strconv.Atoi(ageRaw)
	if err != nil {
		return errors.New("age must be int")
	}
	if age < 0 {
		return errors.New("age must be >= 0")
	}
	if age > 128 {
		return errors.New("age must be <= 128")
	}
	in.Age = age
	return nil
}

func (srv *MyApi) handlerProfile(w http.ResponseWriter, r *http.Request) {
	query := ""

	if r.Method == http.MethodPost {
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		query = string(bytes)
	} else {
		query = r.URL.RawQuery
	}

	params := &ProfileParams{}
	unpackErr := params.Unpack(query)
	if unpackErr != nil {
		responseJson, _ := json.Marshal(Response{Error: unpackErr.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseJson)
		return
	}

	responseRaw, err := srv.Profile(r.Context(), *params)

	if err != nil {
		switch apiErr := err.(type) {
		case ApiError:
			w.WriteHeader(apiErr.HTTPStatus)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		responseJson, _ := json.Marshal(Response{Error: err.Error()})

		w.Write(responseJson)
		return
	}

	response := Response{Response: responseRaw}
	responseJson, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseJson)
}
func (srv *MyApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	query := ""
	if r.Method != http.MethodPost {
		responseJson, _ := json.Marshal(Response{Error: "bad method"})
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(responseJson)
		return
	}

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	query = string(bytes)

	if r.Header.Get("X-Auth") != "100500" {
		responseJson, _ := json.Marshal(Response{Error: "unauthorized"})
		w.WriteHeader(http.StatusForbidden)
		w.Write(responseJson)
		return
	}

	params := &CreateParams{}
	unpackErr := params.Unpack(query)
	if unpackErr != nil {
		responseJson, _ := json.Marshal(Response{Error: unpackErr.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseJson)
		return
	}

	responseRaw, err := srv.Create(r.Context(), *params)

	if err != nil {
		switch apiErr := err.(type) {
		case ApiError:
			w.WriteHeader(apiErr.HTTPStatus)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		responseJson, _ := json.Marshal(Response{Error: err.Error()})

		w.Write(responseJson)
		return
	}

	response := Response{Response: responseRaw}
	responseJson, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseJson)
}
func (in *OtherCreateParams) Unpack(query string) error {
	values, err := url.ParseQuery(query)
	if err != nil {
		return err
	}

	username := values.Get("username")
	if len(username) == 0 {
		return errors.New("username must be not empty")
	}

	if len(username) < 3 {
		return errors.New("username len must be >= 3")
	}
	in.Username = username
	name := values.Get("account_name")
	in.Name = name
	class := values.Get("class")
	if class == "" {
		class = "warrior"
	}

	if class != "warrior" && class != "sorcerer" && class != "rouge" {
		return errors.New("class must be one of [warrior, sorcerer, rouge]")
	}
	in.Class = class
	levelRaw := values.Get("level")
	level, err := strconv.Atoi(levelRaw)
	if err != nil {
		return errors.New("level must be int")
	}
	if level < 1 {
		return errors.New("level must be >= 1")
	}
	if level > 50 {
		return errors.New("level must be <= 50")
	}
	in.Level = level
	return nil
}

func (srv *OtherApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	query := ""
	if r.Method != http.MethodPost {
		responseJson, _ := json.Marshal(Response{Error: "bad method"})
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(responseJson)
		return
	}

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	query = string(bytes)

	if r.Header.Get("X-Auth") != "100500" {
		responseJson, _ := json.Marshal(Response{Error: "unauthorized"})
		w.WriteHeader(http.StatusForbidden)
		w.Write(responseJson)
		return
	}

	params := &OtherCreateParams{}
	unpackErr := params.Unpack(query)
	if unpackErr != nil {
		responseJson, _ := json.Marshal(Response{Error: unpackErr.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseJson)
		return
	}

	responseRaw, err := srv.Create(r.Context(), *params)

	if err != nil {
		switch apiErr := err.(type) {
		case ApiError:
			w.WriteHeader(apiErr.HTTPStatus)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		responseJson, _ := json.Marshal(Response{Error: err.Error()})

		w.Write(responseJson)
		return
	}

	response := Response{Response: responseRaw}
	responseJson, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseJson)
}
func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		srv.handlerCreate(w, r)
	default:
		responseJson, _ := json.Marshal(Response{Error: "unknown method"})
		w.WriteHeader(http.StatusNotFound)
		w.Write(responseJson)
	}
}
func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		srv.handlerProfile(w, r)
	case "/user/create":
		srv.handlerCreate(w, r)
	default:
		responseJson, _ := json.Marshal(Response{Error: "unknown method"})
		w.WriteHeader(http.StatusNotFound)
		w.Write(responseJson)
	}
}
