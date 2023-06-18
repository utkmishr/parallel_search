package main

import (
	"fmt"
	"parallelsearch/app/psearch"
)

func main() {
	fmt.Println("here")
	psearch.MultiRead("/Users/utkarsh/dev/projects/parallel_search/cmd/test-file.txt", "foo")

}
