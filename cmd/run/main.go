package main

import (
	"log"
	"promoter/pkg/tg"
	"promoter/pkg/tg/auth"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	var client = tg.CreateClient()
	defer tg.DestoryClient(client)

	var res, err = auth.GetAuthorizationState(client)
	if err != nil {
		panic(err)
	}
	log.Println(spew.Sdump(res))
}
