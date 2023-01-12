package s3

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awscreds "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// minioProvider is a shim that implements the Minio `Provider` interface
// for an AWS credential
type minioProvider struct {
	creds *awscreds.Credentials
}

func assumeRoleCredentials(roleARN string, roleSessionName string, tags map[string]string) (credentials.Provider, error) {
	sess, err := session.NewSession()
	if err != nil {
		return &minioProvider{}, err
	}

	stsTags := make([]*sts.Tag, 0, len(tags))
	for k, v := range tags {
		tag := sts.Tag{Key: aws.String(k), Value: aws.String(v)}
		stsTags = append(stsTags, &tag)
	}

	creds := stscreds.NewCredentials(
		sess,
		roleARN,
		func(p *stscreds.AssumeRoleProvider) {
			p.Tags = stsTags
			p.RoleSessionName = roleSessionName
			p.Duration = 1 * time.Hour
		})
	return &minioProvider{creds: creds}, nil
}

func (mp *minioProvider) Retrieve() (credentials.Value, error) {
	if mp.creds == nil {
		return credentials.Value{}, nil
	}

	v, err := mp.creds.Get()
	if err != nil {
		return credentials.Value{}, err
	}

	return credentials.Value{
		AccessKeyID:     v.AccessKeyID,
		SecretAccessKey: v.SecretAccessKey,
		SessionToken:    v.SessionToken,
		SignerType:      credentials.SignatureV4,
	}, nil
}

func (mp *minioProvider) IsExpired() bool {
	return mp.creds.IsExpired()
}
