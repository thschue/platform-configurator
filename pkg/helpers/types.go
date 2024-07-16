package helpers

type TlsConfig struct {
	InsecureSkipVerify bool `yaml:"insecureSkipVerify"`
}

type Credentials struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
