package main

import (
	"log"

	"github.com/MoonSHRD/TelegramNFTWizard/bot"
	"github.com/MoonSHRD/TelegramNFTWizard/config"
	"github.com/caarlos0/env/v6"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	cfg := config.Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}

	b, err := bot.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	b.Start()
}
