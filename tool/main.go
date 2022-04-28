package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main(){
	var wg sync.WaitGroup
	request := func(url string){
		defer wg.Done()
		for {
			_, err:=http.DefaultClient.Get(url)
			if err != nil {
				fmt.Println(err)
			}
			time.Sleep(2 *time.Second)
		}
	}

	for i:=0;i<100;i++{
		wg.Add(1)
		go request("http://argocd-o11y-demo.default.svc.cluster.local:8080")
	}
	wg.Wait()

}
