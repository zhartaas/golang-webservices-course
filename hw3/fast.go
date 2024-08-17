package main

import (
	"fmt"
	"github.com/mailru/easyjson"
	"io"
	"os"
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

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	fileContents, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	seenBrowsers := []string{}
	uniqueBrowsers := 0
	foundUsers := ""

	lines := strings.Split(string(fileContents), "\n")

	users := []User{}
	for _, line := range lines {
		user := User{}
		// json.Unmarshal takes too long (510 ms)
		// optimie it using easyjson
		err := easyjson.Unmarshal([]byte(line), &user)
		if err != nil {
			panic(err)
		}
		users = append(users, user)
	}

	for i, user := range users {

		isAndroid := false
		isMSIE := false

		for _, browser := range user.Browsers {
			// regexp too slow so use strings.Contains
			ok := strings.Contains(browser, "Android")
			if ok {
				isAndroid = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}
		for _, browser := range user.Browsers {

			ok := strings.Contains(browser, "MSIE")
			if ok {
				isMSIE = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		//email := r.ReplaceAllString(user.Email, " [at] ")
		// using strings.Replace() instead of using regexp
		email := strings.Replace(user.Email, "@", " [at] ", 1)
		foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email)
	}
	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}

//func FastSearch(out io.Writer) {
//	file, err := os.Open(filePath)
//	if err != nil {
//		panic(err)
//	}
//
//	fileContents, err := io.ReadAll(file)
//	if err != nil {
//		panic(err)
//	}
//	r := regexp.MustCompile("@")
//
//	seenBrowsers := []string{}
//	uniqueBrowsers := 0
//	foundUsers := ""
//
//	lines := strings.Split(string(fileContents), "\n")
//
//	users := make([]map[string]interface{}, 0)
//	for _, line := range lines {
//		user := make(map[string]interface{})
//		// fmt.Printf("%v %v\n", err, line)
//		err := json.Unmarshal([]byte(line), &user)
//		if err != nil {
//			panic(err)
//		}
//		users = append(users, user)
//	}
//
//	for i, user := range users {
//
//		isAndroid := false
//		isMSIE := false
//
//		browsers, ok := user["browsers"].([]interface{})
//		if !ok {
//			// log.Println("cant cast browsers")
//			continue
//		}
//
//		for _, browserRaw := range browsers {
//			browser, ok := browserRaw.(string)
//			if !ok {
//				// log.Println("cant cast browser to string")
//				continue
//			}
//			if ok, err := regexp.MatchString("Android", browser); ok && err == nil {
//				isAndroid = true
//				notSeenBefore := true
//				for _, item := range seenBrowsers {
//					if item == browser {
//						notSeenBefore = false
//					}
//				}
//				if notSeenBefore {
//					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
//					seenBrowsers = append(seenBrowsers, browser)
//					uniqueBrowsers++
//				}
//			}
//		}
//
//		for _, browserRaw := range browsers {
//			browser, ok := browserRaw.(string)
//			if !ok {
//				// log.Println("cant cast browser to string")
//				continue
//			}
//			if ok, err := regexp.MatchString("MSIE", browser); ok && err == nil {
//				isMSIE = true
//				notSeenBefore := true
//				for _, item := range seenBrowsers {
//					if item == browser {
//						notSeenBefore = false
//					}
//				}
//				if notSeenBefore {
//					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
//					seenBrowsers = append(seenBrowsers, browser)
//					uniqueBrowsers++
//				}
//			}
//		}
//
//		if !(isAndroid && isMSIE) {
//			continue
//		}
//
//		// log.Println("Android and MSIE user:", user["name"], user["email"])
//		email := r.ReplaceAllString(user["email"].(string), " [at] ")
//		foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user["name"], email)
//	}
//
//	fmt.Fprintln(out, "found users:\n"+foundUsers)
//	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
//}
