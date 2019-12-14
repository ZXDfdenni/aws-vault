package vault

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/sts"
)

// AssumeRoleProvider retrieves temporary credentials using STS AssumeRole
type AssumeRoleProvider struct {
	StsClient       *sts.STS
	RoleARN         string
	RoleSessionName string
	ExternalID      string
	Duration        time.Duration
	Mfa
	credentials.Expiry
}

// Retrieve returns temporary credentials using STS AssumeRole
func (p *AssumeRoleProvider) Retrieve() (credentials.Value, error) {
	log.Println("Getting credentials with AssumeRole")

	role, err := p.assumeRole()
	if err != nil {
		return credentials.Value{}, err
	}

	log.Printf("Using role %s, expires in %s", formatKeyForDisplay(*role.AccessKeyId), time.Until(*role.Expiration).String())

	p.SetExpiration(*role.Expiration, DefaultExpirationWindow)
	return credentials.Value{
		AccessKeyID:     *role.AccessKeyId,
		SecretAccessKey: *role.SecretAccessKey,
		SessionToken:    *role.SessionToken,
	}, nil
}

func (p *AssumeRoleProvider) roleSessionName() string {
	if p.RoleSessionName == "" {
		// Try to work out a role name that will hopefully end up unique.
		return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}

	return p.RoleSessionName
}

// assumeRoleFromCreds uses the master credentials to assume a role
func (p *AssumeRoleProvider) assumeRole() (*sts.Credentials, error) {
	var err error

	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(p.RoleARN),
		RoleSessionName: aws.String(p.roleSessionName()),
		DurationSeconds: aws.Int64(int64(p.Duration.Seconds())),
	}

	if p.ExternalID != "" {
		input.ExternalId = aws.String(p.ExternalID)
	}

	// if we don't have a session, we need to include MFA token in the AssumeRole call
	if p.MfaSerial != "" {
		input.SerialNumber = aws.String(p.MfaSerial)
		input.TokenCode, err = p.GetMfaToken()
		if err != nil {
			return nil, err
		}
	}

	resp, err := p.StsClient.AssumeRole(input)
	if err != nil {
		return nil, err
	}

	return resp.Credentials, nil
}
