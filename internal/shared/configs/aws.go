package configs

type AWSConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Endpoint        string
}

func (a *AWSConfig) NewAWSConfig() *AWSConfig {
	return &AWSConfig{
		AccessKeyID:     a.AccessKeyID,
		SecretAccessKey: a.SecretAccessKey,
		Region:          a.Region,
		Endpoint:        a.Endpoint,
	}
}
