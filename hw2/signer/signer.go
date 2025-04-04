package main

import (
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	out := make(chan interface{})
	wg := &sync.WaitGroup{}
	for _, function := range jobs {
		in := out
		out = make(chan interface{})
		wg.Add(1)
		go func(fn job, in, out chan interface{}) {
			defer wg.Done()
			fn(in, out)
			close(out)
		}(function, in, out)
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	md5chan := make(chan struct{}, 1)
	wgg := &sync.WaitGroup{}
	for inp := range in {
		//fmt.Println(inp, "SingleHash data", inp)
		strInp := strconv.Itoa(inp.(int))
		wgg.Add(1)
		go func(strInput string) {
			defer wgg.Done()
			wg := &sync.WaitGroup{}
			hashCrc32 := ""

			wg.Add(1)
			go func(input string) {
				defer wg.Done()
				hashCrc32 = DataSignerCrc32(input)
				//fmt.Println(input, "SingleHash", "crc32(data)", hashCrc32)
			}(strInput)
			hashMd5 := ""
			md5chan <- struct{}{}
			hashMd5 = DataSignerMd5(strInput)
			<-md5chan
			//fmt.Println(strInput, "SingleHash", "md5(data)", hashMd5)
			hashCrc32Md5 := DataSignerCrc32(hashMd5)
			//fmt.Println(strInput, "SingleHash", "crc32(md5(data))", hashMd5)
			wg.Wait()
			out <- hashCrc32 + "~" + hashCrc32Md5
		}(strInp)
	}
	wgg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for inp := range in {
		strInput := inp.(string)
		wg.Add(1)
		go func(input string) {
			defer wg.Done()
			m := &sync.Map{}
			wgLoop := &sync.WaitGroup{}
			for i := 0; i < 6; i++ {
				wgLoop.Add(1)
				go func(i int, strInput string) {
					defer wgLoop.Done()
					hashCrc32 := DataSignerCrc32(strconv.Itoa(i) + strInput)
					m.Store(i, hashCrc32)

				}(i, strInput)
			}
			wgLoop.Wait()
			res := ""
			for i := 0; i < 6; i++ {
				str, ok := m.Load(i)
				if !ok {
					fmt.Println(i)
					log.Fatal(ok)
				}
				res += str.(string)
			}
			out <- res
		}(strInput)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {

	res := make([]string, 0)
	for inp := range in {
		res = append(res, inp.(string))
	}

	slices.Sort(res)
	out <- strings.Join(res, "_")
}
