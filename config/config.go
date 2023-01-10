package config

import "github.com/MoonSHRD/TelegramNFTWizard/pkg/blockchain"

type Config struct {
	blockchain.Config
	Token        string `env:"TOKEN,notEmpty"`
	DatabasePath string `env:"DATABASE_PATH" envDefault:"./data"`
}
