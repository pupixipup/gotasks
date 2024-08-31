package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

const LOOP_SIZE = 6

func doJob(job job, in, out chan interface{}, wg *sync.WaitGroup) {
	job(in, out)
	wg.Done()
	close(out)
}

func ExecutePipeline(jobs ...job) {
	var wg sync.WaitGroup
	var in = make(chan interface{})
	var out = make(chan interface{})
	for _, job := range jobs {
		in = out
		out = make(chan interface{})
		wg.Add(1)
		go doJob(job, in, out, &wg)
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	for value := range in {
		m5 := DataSignerMd5(strconv.Itoa(value.(int)))
		wg.Add(1)
		go MakeSingleHash(in, out, value, &wg, m5)
	}
	wg.Wait()
}

func MakeSingleHash(in, out chan interface{}, value interface{}, wg *sync.WaitGroup, m5 string) {
	strValue := strconv.Itoa(value.(int))
	chan1 := make(chan string)
	chan2 := make(chan string)
	go func() {
		chan1 <- DataSignerCrc32(strValue)
	}()
	go func() {
		chan2 <- DataSignerCrc32(m5)
	}()
	out <- (<-chan1) + "~" + (<-chan2)
	wg.Done()
}

func MultiHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	for value := range in {
		wg.Add(1)
		go MakeMultiHash(in, out, value, &wg)
	}
	wg.Wait()
}

func MakeMultiHash(in, out chan interface{}, value interface{}, wg *sync.WaitGroup) {
	var wgIn sync.WaitGroup
	hashes := make([]string, LOOP_SIZE)
	for i := 0; i <= 5; i++ {
		strIndex := strconv.Itoa(i)
		index := i
		wgIn.Add(1)
		go func() {
			piece := DataSignerCrc32(strIndex + value.(string))
			hashes[index] = piece
			wgIn.Done()
		}()
	}
	wgIn.Wait()
	out <- strings.Join(hashes, "")
	wg.Done()
}

func CombineResults(in, out chan interface{}) {
	var result []string
	for value := range in {
		result = append(result, value.(string))
	}
	sort.Strings(result)
	out <- strings.Join(result, "_")
}
