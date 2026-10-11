package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/SpaceTent/EnvSetter/pkg/envvaultclient"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	options := envvaultclient.FromEnv(envvaultclient.Options{})
	vars, err := envvaultclient.Load(ctx, options)
	if errors.Is(err, envvaultclient.ErrWrongKey) {
		log.Fatal("the configured key cannot open this EnvVault secret")
	}
	if err != nil {
		log.Fatal(err)
	}

	// Use values without logging them. This example prints only their names.
	for name := range vars {
		fmt.Println(name)
	}
}
