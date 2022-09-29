package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"

	"github.com/federicoleon/golang-microservices/src/api/domain/repositories"
	"github.com/federicoleon/golang-microservices/src/api/services"
	"github.com/federicoleon/golang-microservices/src/api/utils/errors"
)

var (
	success = make(map[string]string, 0)
	failed  = make(map[string]errors.ApiError, 0)
)

type createRepoResult struct {
	Request repositories.CreateRepoRequest
	Result  *repositories.CreateRepoResponse
	Error   errors.ApiError
}

func getRequests() []repositories.CreateRepoRequest {
	result := make([]repositories.CreateRepoRequest, 0)

	file, err := os.Open("/Users/fleon/Desktop/requests/requests.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		request := repositories.CreateRepoRequest{
			Name: line,
		}
		result = append(result, request)
	}
	return result
}

func main() {
	requests := getRequests()

	fmt.Println(fmt.Sprintf("about to process %d requests", len(requests)))

	input := make(chan createRepoResult)
	// throtelling limit on the github due to rate limit
	buffer := make(chan bool, 10)
	var wg sync.WaitGroup

	go handleResults(&wg, input)
	// create 10 request and wait for the buffered channel to be read by some one
	// first 10 go routines are created and blocks on 11th one
	// if buffered channels were not used here it keep creating 36000(entiries in the text file) go routines, which wont
	// be a issue, but there will be a rate limit on github.com
	// find the rate limit value calculate and keet the value for buffered channel accordingly
	for _, request := range requests {
		buffer <- true
		wg.Add(1)
		go createRepo(buffer, input, request)
	}

	wg.Wait()
	close(input)

	// Now you can write success and failed maps to disk or notify them via email or anything you need to do.
}

func handleResults(wg *sync.WaitGroup, input chan createRepoResult) {
	for result := range input {
		if result.Error != nil {
			failed[result.Request.Name] = result.Error
		} else {
			success[result.Request.Name] = result.Result.Name
		}
		wg.Done()
	}
}

func createRepo(buffer chan bool, output chan createRepoResult, request repositories.CreateRepoRequest) {
	result, err := services.RepositoryService.CreateRepo("your_client_id", request)

	output <- createRepoResult{
		Request: request,
		Result:  result,
		Error:   err,
	}

	// buffer channel read, to make wake way for other goroutines to be created
	// if not read the buffer channelon L:59 will be blocked
	// this function is run on a go routine to make a request to github.com individually
	// once the go routine complets, it reads from buffer making it empty, and L:59 adds another makes the buffer full
	// blocks once again until it being read by other goroutines.
	<-buffer
}
