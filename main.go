package main

import (
	"fmt"
	"strings"
)

type User struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"company"`
	Country  string   `json:"country"`
	Email    string   `json:"email"`
	Job      string   `json:"job"`
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
}

func main() {
	res := "29568666068035183841425683795340791879727309630931025356555_4958044192186797981418233587017209679042592862002427381542"

	str := strings.Split(res, "_")
	fmt.Println(str, str[0], str[1])
}
