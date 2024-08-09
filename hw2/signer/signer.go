package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(freeFlowJobs ...job) {
	out := make(chan interface{})
	wg := &sync.WaitGroup{}
	for _, inputJob := range freeFlowJobs {
		in := out
		out = make(chan interface{})
		wg.Add(1)
		go func(pipeline job, in, out chan interface{}) {
			defer wg.Done()
			pipeline(in, out)
			close(out)
		}(inputJob, in, out)
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	globalWG := &sync.WaitGroup{}
	md5chan := make(chan struct{}, 1)
	for input := range in {
		//now := time.Now()
		//fmt.Println(input)
		data := strconv.Itoa(input.(int))
		globalWG.Add(1)
		go func() {
			defer globalWG.Done()
			wg := &sync.WaitGroup{}

			crc32data := ""

			wg.Add(1)
			go func() {
				defer wg.Done()
				crc32data = DataSignerCrc32(data)
			}()

			md5data := ""
			crc32md5data := ""

			wg.Add(1)
			go func(ch chan struct{}) {
				md5chan <- struct{}{}
				defer wg.Done()
				md5data = DataSignerMd5(data)
				<-md5chan
				crc32md5data = DataSignerCrc32(md5data)
				//fmt.Println(md5data, crc32md5data)
			}(md5chan)

			wg.Wait()
			result := fmt.Sprintf("%s~%s", crc32data, crc32md5data)
			//fmt.Println(time.Now().Format(time.TimeOnly), "SingleHash: ", input, time.Since(now))
			out <- result
		}()
	}
	globalWG.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for input := range in {
		data := input.(string)
		res := ""
		crc32data := &sync.Map{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			wg := &sync.WaitGroup{}
			for i := 0; i < 6; i++ {
				wg.Add(1)
				go func(th int) {
					defer wg.Done()
					thData := fmt.Sprintf("%s%s", strconv.Itoa(th), data)
					crc32data.Store(th, DataSignerCrc32(thData))
				}(i)
			}
			wg.Wait()

			//fmt.Println(crc32data)
			for i := 0; i < 6; i++ {
				v, ok := crc32data.Load(i)
				if !ok {
					fmt.Println("error loading ", i)
					return
				}
				res += v.(string)
			}
			out <- res
		}()
	}
	wg.Wait()
}

//func MultiHash(in, out chan interface{}) {
//	for input := range in {
//		now := time.Now()
//		data := input.(string)
//		res := ""
//		crc32data := &sync.Map{}
//		wg := &sync.WaitGroup{}
//		wg.Add(1)
//		go func() {
//			defer wg.Done()
//			for i := 0; i < 6; i++ {
//				wg.Add(1)
//				go func(th int) {
//					defer wg.Done()
//					thData := fmt.Sprintf("%s%s", strconv.Itoa(th), data)
//					crc32data.Store(th, DataSignerCrc32(thData))
//				}(i)
//				//fmt.Printf("%s crc32(th+step1)) %v %s\n", prefix, i, crc32data)
//			}
//		}()
//		wg.Wait()
//		for i := 0; i < 6; i++ {
//			v, ok := crc32data.Load(i)
//			if !ok {
//				fmt.Println(1)
//				return
//			}
//			res += v.(string)
//		}
//		fmt.Println("1231231")
//		//fmt.Printf("%s result: %s\n\n", prefix, res)
//		out <- res
//		fmt.Printf("MultiHash %v, %s \n\n", input, time.Since(now))
//	}
//}

func CombineResults(in, out chan interface{}) {
	res := make([]string, 0)
	for input := range in {
		res = append(res, input.(string))
	}

	sort.Strings(res)

	out <- strings.Join(res, "_")
}
