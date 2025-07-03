package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Acrylic125/webhook-ingest-ws/utils"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

const (
	EnvProduction = "prod"
	EnvStaging    = "staging"
	EnvDev        = "dev"
	EnvLocal      = "local"
)

type Settings struct {
	Secrets *Secrets
	Configs *Configs
}

type Configs struct {
	WebhookTargetUrl string `json:",omitempty" validate:"required" default:"staging-api.limbolabs.xyz/watchlist"`
}

type Secrets struct {
	CodexToken string `json:",omitempty" validate:"required"`
}

var (
	settings = Settings{}
	env      = EnvLocal
	logger   = log.Logger
)

func Env() string {
	return env
}

func Get() Settings {
	return settings
}

func Init(envp string) {
	logger = log.With().
		Str("env", envp).
		Str("service", "settings_init").
		Logger()

	logger.Trace().Msg("initializing settings")

	env = envp
	if env == "" {
		env = EnvLocal
		logger.Warn().Msg("GO_ENV value is missing, setting environment as - " + env)
	}
	logger.Info().Msg("running with environment - " + env)

	logger.Trace().Msg("loading settings")
	if err := LoadSettingsByEnv(); err != nil {
		// If there's err in loading settings, blow-up!
		logger.Fatal().Err(err).Msg("failed to load ENV")
	}

	log.Info().Msg("settings initialized")
}

func LoadSettingsByEnv() error {
	secrets, err := getSecrets()
	if err != nil {
		return fmt.Errorf("failed to load secrets: %w", err)
	}
	settings.Secrets = secrets

	configs, err := getConfigs()
	if err != nil {
		return fmt.Errorf("failed to load configs: %w", err)
	}
	settings.Configs = configs

	return nil
}

func getSecrets() (*Secrets, error) {
	result := &Secrets{}
	var secretsContent []byte
	if env != EnvProduction {
		secretsFile := fmt.Sprintf("internal/settings/secrets_%s.json", env)
		content, err := os.ReadFile(filepath.Clean(secretsFile))
		if err != nil {
			return nil, fmt.Errorf("error while reading secrets file [%s]: %w", secretsFile, err)
		}
		secretsContent = content
	} else {
		// If the environment is production, we will get the secrets from the injected env
		injectedSecrets := os.Getenv("SECRETS")
		secretsContent = []byte(injectedSecrets)
	}

	if jsonErr := json.Unmarshal(secretsContent, result); jsonErr != nil {
		return nil, fmt.Errorf(
			"error while parsing secretsContent for env[%s]: %w",
			env,
			jsonErr,
		)
	}

	if err := validator.New().Struct(result); err != nil {
		return nil, fmt.Errorf("secrets content struct validation failed: %w", err)
	}

	return result, nil
}

func getConfigs() (*Configs, error) {
	result := &Configs{}
	var configsContent []byte
	if env != EnvProduction {
		configFile := fmt.Sprintf("internal/settings/config_%s.json", env)
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			configsContent = []byte("{}")
		} else {
			content, err := os.ReadFile(filepath.Clean(configFile))
			if err != nil {
				return nil, fmt.Errorf("error while reading config file [%s]: %w", configFile, err)
			}

			configsContent = content
		}
	} else {
		// If the environment is production, we will get the config from the injected env
		injectedConfigs := os.Getenv("CONFIGS")
		if injectedConfigs == "" {
			configsContent = []byte("{}")
		} else {
			configsContent = []byte(injectedConfigs)
		}
	}

	if jsonErr := json.Unmarshal(configsContent, result); jsonErr != nil {
		return nil, fmt.Errorf(
			"error while parsing configsContent for env[%s]: %w",
			configsContent,
			jsonErr,
		)
	}

	// Map the default values
	if err := utils.MapDefaults(result); err != nil {
		return nil, fmt.Errorf("failed to map default values: %w", err)
	}

	if err := validator.New().Struct(result); err != nil {
		return nil, fmt.Errorf("failed to validate struct: %w", err)
	}

	return result, nil
}
