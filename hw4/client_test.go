package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// код писать тут

type XML struct {
	XMLName xml.Name `xml:"root"`
	Text    string   `xml:",chardata"`
	Row     []struct {
		ID        int    `xml:"id"`
		FirstName string `xml:"first_name"`
		LastName  string `xml:"last_name"`
		Age       int    `xml:"age"`
		About     string `xml:"about"`
		Gender    string `xml:"gender"`
	} `xml:"row"`
}

type TestCase struct {
	AccessToken string
	SearchRequest
	SearchResponse
	IsError bool
	Error   error
}

func TestSearchServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	cases := []TestCase{
		{
			AccessToken: "1",
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "Jordan",
				OrderField: "",
				OrderBy:    OrderByAsIs,
			},
			SearchResponse: SearchResponse{
				Users: []User{
					{
						Id:     8,
						Name:   "Glenn Jordan",
						Age:    29,
						About:  "Duis reprehenderit sit velit exercitation non aliqua magna quis ad excepteur anim. Eu cillum cupidatat sit magna cillum irure occaecat sunt officia officia deserunt irure. Cupidatat dolor cupidatat ipsum minim consequat Lorem adipisicing. Labore fugiat cupidatat nostrud voluptate ea eu pariatur non. Ipsum quis occaecat irure amet esse eu fugiat deserunt incididunt Lorem esse duis occaecat mollit.",
						Gender: "male",
					},
				},
				NextPage: false,
			},
			IsError: false,
		},
		{
			AccessToken: "1",
			SearchRequest: SearchRequest{
				Limit:      -1,
				Offset:     0,
				Query:      "Jordan",
				OrderField: "",
				OrderBy:    OrderByAsIs,
			},
			IsError: true,
			Error:   fmt.Errorf("limit must be > 0"),
		},
		{
			AccessToken: "1",
			SearchRequest: SearchRequest{
				Limit:      30,
				Offset:     0,
				Query:      "Jo",
				OrderField: "Age",
				OrderBy:    OrderByDesc,
			},
			SearchResponse: SearchResponse{
				Users: []User{
					{
						Id:     21,
						Name:   "Johns Whitney",
						Age:    26,
						About:  "Elit sunt exercitation incididunt est ea quis do ad magna. Commodo laboris nisi aliqua eu incididunt eu irure. Labore ullamco quis deserunt non cupidatat sint aute in incididunt deserunt elit velit. Duis est mollit veniam aliquip. Nulla sunt veniam anim et sint dolore.",
						Gender: "male",
					},
					{
						Id:     8,
						Name:   "Glenn Jordan",
						Age:    29,
						About:  "Duis reprehenderit sit velit exercitation non aliqua magna quis ad excepteur anim. Eu cillum cupidatat sit magna cillum irure occaecat sunt officia officia deserunt irure. Cupidatat dolor cupidatat ipsum minim consequat Lorem adipisicing. Labore fugiat cupidatat nostrud voluptate ea eu pariatur non. Ipsum quis occaecat irure amet esse eu fugiat deserunt incididunt Lorem esse duis occaecat mollit.",
						Gender: "male",
					},
				},
			},
		},
		{
			AccessToken: "1",
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     -1,
				Query:      "Jordan",
				OrderField: "",
				OrderBy:    OrderByAsIs,
			},
			IsError: true,
			Error:   fmt.Errorf("offset must be > 0"),
		},
		{
			AccessToken: "",
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "Jordan",
				OrderField: "",
				OrderBy:    OrderByAsIs,
			},
			IsError: true,
			Error:   fmt.Errorf("Bad AccessToken"),
		},
		{
			AccessToken: "1",
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "Jordan",
				OrderField: "Ag",
				OrderBy:    OrderByAsIs,
			},
			IsError: true,
			Error:   fmt.Errorf("OrderFeld Ag invalid"),
		},
		{
			AccessToken: "1",
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "Jo",
				OrderField: "Age",
				OrderBy:    OrderByDesc,
			},
			SearchResponse: SearchResponse{
				Users: []User{
					{
						Id:     21,
						Name:   "Johns Whitney",
						Age:    26,
						About:  "Elit sunt exercitation incididunt est ea quis do ad magna. Commodo laboris nisi aliqua eu incididunt eu irure. Labore ullamco quis deserunt non cupidatat sint aute in incididunt deserunt elit velit. Duis est mollit veniam aliquip. Nulla sunt veniam anim et sint dolore.",
						Gender: "male",
					},
				},
				NextPage: true,
			},
		},
		{
			AccessToken: "11",
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "Jordan",
				OrderField: "",
				OrderBy:    OrderByAsIs,
			},
			IsError: true,
			Error:   fmt.Errorf("timeout for limit=%d&offset=%d&order_by=%d&order_field=&query=%s", 2, 0, 0, "Jordan"),
		},
		{
			AccessToken: "1",
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "query",
				OrderField: "",
				OrderBy:    OrderByAsIs,
			},
			IsError: true,
			Error:   fmt.Errorf("unknown error Get \"%s?limit=2&offset=0&order_by=0&order_field=&query=query\": EOF", ts.URL),
		},
	}

	for caseNum, item := range cases {
		client := &SearchClient{
			AccessToken: item.AccessToken,
			URL:         ts.URL,
		}

		result, err := client.FindUsers(item.SearchRequest)
		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err != nil && err.Error() != item.Error.Error() {
			t.Errorf("[%d] wrong error, expected %v, got %v", caseNum, item.Error, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if err == nil && !reflect.DeepEqual(result, &item.SearchResponse) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.SearchResponse, result)
		}

	}

}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	accessToken := r.Header.Get("AccessToken")
	if len(accessToken) == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	time.Sleep(time.Millisecond * 500 * time.Duration(len(accessToken)))
	query := r.FormValue("query")
	if query == "query" {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			fmt.Println("Hijack error:", err)
			return
		}
		conn.Close()
	}
	orderField := r.FormValue("order_field")
	if orderField != "" && orderField != "Name" && orderField != "Id" && orderField != "Age" {
		w.WriteHeader(http.StatusBadRequest)
		reqErr := SearchErrorResponse{Error: "ErrorBadOrderField"}
		json, err := json.Marshal(reqErr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(json)
		return
	}
	orderBy, err := strconv.Atoi(r.FormValue("order_by"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error order by"))
		return
	}

	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil || limit < 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("limit error"))
		return
	}
	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("offset error"))
		return
	}

	req := SearchRequest{
		Limit:      limit,
		Offset:     offset,
		Query:      query,
		OrderField: orderField,
		OrderBy:    orderBy,
	}

	res, err := findUsers(req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	resJson, err := json.Marshal(res)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(resJson)
}

func findUsers(params SearchRequest) ([]User, error) {
	f, err := os.Open("dataset.xml")
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	data := &XML{}
	if err := xml.Unmarshal(bytes, data); err != nil {
		return nil, err
	}

	users := []User{}

	for _, user := range data.Row {
		name := user.FirstName + " " + user.LastName
		if strings.Contains(name, params.Query) || strings.Contains(user.About, params.Query) {
			foundUser := User{
				Id:     user.ID,
				Name:   name,
				Age:    user.Age,
				About:  strings.TrimSpace(user.About),
				Gender: user.Gender,
			}
			users = append(users, foundUser)
		}
	}

	if params.OrderField == "Age" {
		slices.SortStableFunc(users, func(a, b User) int {
			return (a.Age - b.Age) * params.OrderBy
		})
	} else if params.OrderField == "Name" || params.OrderField == "" {
		slices.SortStableFunc(users, func(a, b User) int {
			return strings.Compare(a.Name, b.Name) * params.OrderBy
		})
	}

	params.Limit += params.Offset
	params.Limit = int(math.Min(float64(params.Limit), float64(len(users))))
	return users[params.Offset:params.Limit], nil
}
