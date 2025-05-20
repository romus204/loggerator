package config

import (
	"log"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	Telegram Telegram `yaml:"telegram"` // telegram chat id
	Kube     Kube     `yaml:"kube"`     // kube
}

type Telegram struct {
	Token  string         `yaml:"token"` // telegram bot token
	Chat   int            `yaml:"chat"`  // cat ids
	Topics map[string]int `yaml:"topics"`
}

type Kube struct {
	Target       []Target       `yaml:"target"`    // pod and containers name
	KubeConfig   string         `yaml:"config"`    // path to kube config
	Namespace    string         `yaml:"namespace"` // kube namespace
	Filter       []string       `yaml:"filter"`
	Replacements []Replacements `yaml:"replacements"`
	Rest         *rest.Config
}

type Target struct {
	Pod       string   `yaml:"pod"`       // pod name
	Container []string `yaml:"container"` // container names
}

type Replacements struct {
	Target      string `yaml:"target"`
	Replacement string `yaml:"replacement"`
}

// substituteEnvVars replaces ${ENV_VAR} with the value of the environment variable
func substituteEnvVars(value string) string {
	re := regexp.MustCompile(`\${([^}]+)}`)
	return re.ReplaceAllStringFunc(value, func(match string) string {
		envVar := match[2 : len(match)-1] // Remove ${ and }
		if value, exists := os.LookupEnv(envVar); exists {
			return value
		}
		return match // Return original if env var not found
	})
}

// processEnvVars recursively processes all string fields in the config
func (c *Config) processEnvVars() {
	// Process Telegram fields
	c.Telegram.Token = substituteEnvVars(c.Telegram.Token)

	// Process Kube fields
	c.Kube.KubeConfig = substituteEnvVars(c.Kube.KubeConfig)
	c.Kube.Namespace = substituteEnvVars(c.Kube.Namespace)

	// Process Replacements
	for i := range c.Kube.Replacements {
		c.Kube.Replacements[i].Target = substituteEnvVars(c.Kube.Replacements[i].Target)
		c.Kube.Replacements[i].Replacement = substituteEnvVars(c.Kube.Replacements[i].Replacement)
	}

	// Process Filter
	for i := range c.Kube.Filter {
		c.Kube.Filter[i] = substituteEnvVars(c.Kube.Filter[i])
	}

	// Process Target
	for i := range c.Kube.Target {
		c.Kube.Target[i].Pod = substituteEnvVars(c.Kube.Target[i].Pod)
		for j := range c.Kube.Target[i].Container {
			c.Kube.Target[i].Container[j] = substituteEnvVars(c.Kube.Target[i].Container[j])
		}
	}
}

func (c *Config) setRestKube() {
	if kubeConfigContent := os.Getenv("KUBE_CONFIG_CONTENT"); kubeConfigContent != "" {
		tmpFile, err := os.CreateTemp("", "kubeconfig-*")
		if err != nil {
			log.Fatal("error create temp file")
		}
		defer os.Remove(tmpFile.Name())

		if _, err = tmpFile.WriteString(kubeConfigContent); err != nil {
			log.Fatal("error write in tmp file")
		}
		if err = tmpFile.Close(); err != nil {
			log.Fatal("error close tmp file")
		}

		c.Kube.Rest, err = clientcmd.BuildConfigFromFlags("", tmpFile.Name())
		if err != nil {
			log.Fatal("error create rest")
		}
		return
	}

	rest, err := clientcmd.BuildConfigFromFlags("", c.Kube.KubeConfig)
	if err != nil {
		log.Fatal("error create rest")
	}

	c.Kube.Rest = rest
}

func NewConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	cfg := Config{}

	if decodeErr := yaml.NewDecoder(file).Decode(&cfg); decodeErr != nil {
		return Config{}, decodeErr
	}

	cfg.processEnvVars()
	cfg.setRestKube()

	return cfg, nil
}
