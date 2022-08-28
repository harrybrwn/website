package main

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

func main() {
	master := "rdbmaster"
	sentinels := make([]string, 0)
	for i := 0; i < 3; i++ {
		addr := fmt.Sprintf("sentinel-%d.sentinel.default.svc.cluster.local:26379", i)
		sentinels = append(sentinels, addr)
	}
	fmt.Println("connecting to", master, sentinels)
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:      sentinels,
		MasterName: master,
		Password:   "testbed01",
	})
	ctx := context.Background()
	res, err := client.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(res)

	res, err = client.Get(ctx, "tester").Result()
	fmt.Println("get tester =>", res, err)
	res, err = client.Set(ctx, "tester", "123", redis.KeepTTL).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("set tester 123 =>", res)
	res, err = client.Get(ctx, "tester").Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("get tester =>", res)
}
