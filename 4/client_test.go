package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type Person struct {
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	About     string `xml:"about"`
	Id        int    `xml:"id"`
	Age       int    `xml:"age"`
}

const ACCESS_TOKEN = "test_token"

type Persons struct {
	Persons []Person `xml:"row"`
}

// DONT REUSE SERVER
var ts = httptest.NewServer(http.HandlerFunc(SearchServer))
var test_client = ts.Client()
var searchClient = SearchClient{
	AccessToken: ACCESS_TOKEN,
	URL:         ts.URL,
}

func TestJSONInvalid(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"invalid_json":}`))
	}))
	defer ts.Close()
	var searchClient = SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	searchClient.URL = ts.URL

	req := SearchRequest{
		Limit:      5,
		Offset:     1,
		OrderField: "Name",
		OrderBy:    1,
	}

	_, err := searchClient.FindUsers(req)
	if err == nil {
		t.Errorf("Expected JSON unmarshal error, but got nil")
	} else if !strings.Contains(err.Error(), "cant unpack result json") {
		t.Errorf("Expected error message to contain 'cant unpack result json', but got: %s", err.Error())
	}
}

func TestServer(t *testing.T) {
	req := SearchRequest{
		Limit:      5,
		Offset:     1,
		OrderField: "Name",
		OrderBy:    1,
	}
	_, err := searchClient.FindUsers(req)
	if err != nil {
		t.Errorf("Error during GET request")
	}
}

func TestLimitInvalid(t *testing.T) {
	req := SearchRequest{
		Limit:      -1,
		Offset:     1,
		OrderField: "Name",
		OrderBy:    1,
	}

	_, err := searchClient.FindUsers(req)
	if err == nil {
		t.Errorf("Invalid limit should throw")
	}
}

func TestOrderFieldInvalid(t *testing.T) {
	const ORDER_FIELD = "afllaflf"
	req := SearchRequest{
		Limit:      5,
		Offset:     1,
		OrderField: ORDER_FIELD,
		OrderBy:    1,
	}

	_, err := searchClient.FindUsers(req)
	if err == nil || err.Error() != "OrderFeld "+ORDER_FIELD+" invalid" {
		t.Errorf("Should throw OrderField errow")
	}
}

func TestErrorJSONInvalid(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"invalid_json":}`))
	}))
	defer ts.Close()
	var searchClient = SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}
	searchClient.URL = ts.URL

	req := SearchRequest{
		Limit:      5,
		Offset:     1,
		OrderField: "Name",
		OrderBy:    1,
	}

	_, err := searchClient.FindUsers(req)
	if err == nil {
		t.Errorf("Expected JSON Error unmarshal error, but got nil")
	}
}

func TestMaxLimit(t *testing.T) {
	req := SearchRequest{
		Limit:      99,
		Offset:     1,
		OrderField: "Name",
		OrderBy:    1,
	}

	res, err := searchClient.FindUsers(req)
	if res != nil && len(res.Users) > 25 {
		t.Errorf("Should not exceed limit (25)")
	}
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestOffsetLimit(t *testing.T) {
	req := SearchRequest{
		Limit:      0,
		Offset:     -9,
		OrderField: "Name",
		OrderBy:    1,
	}

	_, err := searchClient.FindUsers(req)
	if err == nil {
		t.Errorf("Should throw with invalid offset")
	}
}

func TestTimeout(t *testing.T) {
	lim := 0
	ofs := 0
	orb := 1
	orf := "Name"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 5)
	}))
	http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = 10 * time.Millisecond

	req := SearchRequest{
		Limit:      lim,
		Offset:     ofs,
		OrderField: orf,
		OrderBy:    orb,
	}

	client := SearchClient{
		URL:         server.URL,
		AccessToken: ACCESS_TOKEN,
	}
	_, error := client.FindUsers(req)
	if !strings.Contains(error.Error(), "timeout for ") {
		t.Errorf("Should throw on timeout")
	}
	http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = 0 // reset timeout
}

func TestInvalidToken(t *testing.T) {
	req := SearchRequest{
		Limit:      3,
		Offset:     1,
		OrderField: "Name",
		OrderBy:    1,
	}

	searchClient := SearchClient{
		AccessToken: "invalid_token",
		URL:         ts.URL,
	}

	_, err := searchClient.FindUsers(req)
	if err == nil {
		t.Errorf("Should throw on invalid token")
	} else if err.Error() != "Bad AccessToken" {
		t.Errorf("Should throw BadAccessToken")
	}
}

func TestInternalError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	client := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	req := SearchRequest{
		Limit:      10,
		Offset:     0,
		Query:      "test",
		OrderField: "Name",
		OrderBy:    OrderByAsc,
	}
	_, err := client.FindUsers(req)

	if err == nil || err.Error() != "SearchServer fatal error" {
		t.Errorf("expected error 'SearchServer fatal error', got %v", err)
	}
}

func TestUniqueThrow(t *testing.T) {
	req := SearchRequest{
		Limit:      1,
		Offset:     10,
		OrderField: "Name",
		OrderBy:    1,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))

	client := SearchClient{
		URL:         "",
		AccessToken: ACCESS_TOKEN,
	}
	server.Close()
	_, err := client.FindUsers(req)

	if err == nil || !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("Should throw fatal error")
	}
}

func TestOrderByInvalid(t *testing.T) {
	req := SearchRequest{
		Limit:      1,
		Offset:     1,
		OrderField: "Name",
		OrderBy:    -2,
	}

	_, err := searchClient.FindUsers(req)

	if err == nil || err.Error() != "unknown bad request error: new error" {
		t.Errorf("should throw on invalid order")
	}
}

func TestNextPage(t *testing.T) {
	req := SearchRequest{
		Limit:      25,
		Offset:     0,
		OrderField: "Name",
		OrderBy:    0,
	}

	const MAX_ITEMS = 35
	for {
		res, err := searchClient.FindUsers(req)
		if err != nil {
			t.Errorf("Should fetch items successfully")
		}
		req.Offset += len(res.Users)
		if !res.NextPage {
			if req.Offset != MAX_ITEMS {
				t.Errorf("NextPage should be false when no items left")
			}
			break
		}
	}
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	if r.Header.Get("AccessToken") != ACCESS_TOKEN {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	hasLimit := len(query["limit"]) > 0
	hasOffset := len(query["offset"]) > 0
	hasQuery := len(query["query"]) > 0
	hasOrderField := len(query["order_field"]) > 0
	hasOrderBy := len(query["order_by"]) > 0
	// Name or About
	orderField := "Name"
	queryStr := ""
	limit := 0
	offset := 0
	orderBy := "0"

	if hasOrderBy {
		orderBy = query["order_by"][0]
	}

	if hasOrderField {
		orderOptions := []string{"Name", "Age", "Id"}
		field := query["order_field"][0]
		if !contains(orderOptions, field) {
			w.WriteHeader(http.StatusBadRequest)
			var errorMap SearchErrorResponse
			errorMap = SearchErrorResponse{
				Error: "ErrorBadOrderField",
			}
			jsonErr, err := json.Marshal(errorMap)
			if err == nil {
				w.Write(jsonErr)
			}
			return
		}
	}

	/* Query */
	if hasQuery {
		queryStr = query["query"][0]
	}

	/* Limit */
	if hasLimit {
		lim, err := strconv.Atoi(query["limit"][0])
		if err != nil {
			hasLimit = false
		} else {
			limit = lim
		}
	}

	/* Offset */
	if hasOffset {
		off, err := strconv.Atoi(query["offset"][0])
		if err != nil {
			hasOffset = false
		} else {
			offset = off
		}
	}

	file, err := os.Open("dataset.xml")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	persons := new(Persons)
	body, err := io.ReadAll(bufio.NewReader(file))
	err = xml.Unmarshal(body, persons)

	var users []Person
	var usersTotal int

	for i, person := range persons.Persons {
		name := person.FirstName + " " + person.LastName
		if i < offset {
			continue
		}
		if hasQuery {
			if strings.Contains(name, queryStr) || strings.Contains(person.About, queryStr) {
				users = append(users, person)
			}
		} else {
			users = append(users, person)
		}
		usersTotal++
		if usersTotal == limit {
			break
		}
	}

	possibleOrderValues := []string{"1", "-1", "0"}
	if !contains(possibleOrderValues, orderBy) {
		w.WriteHeader(http.StatusBadRequest)
		obj, err := json.Marshal(SearchErrorResponse{
			Error: "new error",
		})
		if err != nil {
			panic(err)
		}
		w.Write(obj)
		return
	}

	if orderBy != "0" {
		sort.SliceStable(users, func(i, j int) bool {
			prevIndex := i
			nextIndex := j
			if orderBy == "1" {
				prevIndex = j
				prevIndex = i
			}

			switch orderField {
			case "Id":
				return users[i].Id < users[j].Id
			case "Name":
				namePrev := users[prevIndex].FirstName + " " + users[nextIndex].LastName
				nameNext := users[prevIndex].FirstName + " " + users[nextIndex].LastName
				return namePrev < nameNext
			case "Age":
				return users[prevIndex].Age < users[nextIndex].Age
			default:
				return false // or panic("Invalid field") to handle unexpected fields
			}
		})
	}

	jsonUsers, err := json.Marshal(users)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	_, werr := w.Write(jsonUsers)
	if werr != nil {
		panic(werr)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
