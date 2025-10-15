package awsclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Identity represents the AWS caller identity details.
type Identity struct {
	Account string
	Arn     string
	UserID  string
}

// String returns a human-readable one-line representation.
func (id Identity) String() string {
	if id.Account == "" && id.Arn == "" && id.UserID == "" {
		return ""
	}
	return "AWS Identity: account=" + id.Account + " user=" + id.UserID + "\n" + id.Arn
}

// LoadDefaultConfig loads the AWS configuration with optional region/profile overrides.
// Providing a profile enables AWS SSO or any other shared profile configuration present in ~/.aws/config.
// Returns aws.Config (the canonical SDK config type).
func LoadDefaultConfig(ctx context.Context, region, profile string) (aws.Config, error) {
	var options []func(*awsconfig.LoadOptions) error
	if profile != "" {
		options = append(options, awsconfig.WithSharedConfigProfile(profile))
	} else {
		options = append(options, awsconfig.WithRegion(region))
		options = append(options,
			awsconfig.WithRetryer(func() aws.Retryer { return retry.AddWithMaxAttempts(aws.NopRetryer{}, 1) }))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return aws.Config{}, err
	}
	return cfg, nil
}

// GetIdentity retrieves the current caller identity using STS.
func GetIdentity(ctx context.Context, cfg aws.Config) (Identity, error) {
	client := sts.NewFromConfig(cfg)
	out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		Account: aws.ToString(out.Account), Arn: aws.ToString(out.Arn), UserID: aws.ToString(out.UserId),
	}, nil
}

// AuthError represents an authentication-related error with helpful context.
type AuthError struct {
	Original error
	Message  string
	Hint     string
	Profile  string
	IsSSOErr bool
}

func (e *AuthError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Message)
	if e.Hint != "" {
		sb.WriteString("\n\n")
		sb.WriteString(e.Hint)
	}
	return sb.String()
}

func (e *AuthError) Unwrap() error {
	return e.Original
}

// IsAuthError checks if an error is an authentication error.
func IsAuthError(err error) (*AuthError, bool) {
	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr, true
	}
	return nil, false
}

// ValidateCredentials performs a pre-flight check to validate AWS credentials.
// Returns an AuthError with helpful messages if credentials are invalid or expired.
func ValidateCredentials(ctx context.Context, cfg aws.Config, profile string) error {
	_, err := GetIdentity(ctx, cfg)
	if err != nil {
		return wrapAuthError(err, profile)
	}
	return nil
}

// wrapAuthError wraps AWS SDK errors with helpful authentication error messages.
func wrapAuthError(err error, profile string) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	authErr := &AuthError{
		Original: err,
		Profile:  profile,
	}

	// Check for SSO token expiration
	if strings.Contains(errStr, "failed to refresh cached credentials") ||
		strings.Contains(errStr, "the SSO session has expired") ||
		strings.Contains(errStr, "SSO session is invalid") ||
		strings.Contains(errStr, "token is expired") {
		authErr.IsSSOErr = true
		authErr.Message = "AWS SSO session has expired or is invalid"

		if profile != "" {
			authErr.Hint = fmt.Sprintf("Please run: aws sso login --profile %s", profile)
		} else {
			// Check AWS_PROFILE env var
			if envProfile := os.Getenv("AWS_PROFILE"); envProfile != "" {
				authErr.Hint = fmt.Sprintf("Please run: aws sso login --profile %s", envProfile)
			} else {
				authErr.Hint = "Please run: aws sso login"
			}
		}
		return authErr
	}

	// Check for missing credentials
	if strings.Contains(errStr, "no valid credentials") ||
		strings.Contains(errStr, "NoCredentialProviders") ||
		strings.Contains(errStr, "could not find credentials") {
		authErr.Message = "No AWS credentials found"
		authErr.Hint = buildNoCredentialsHint(profile)
		return authErr
	}

	// Check for access denied / invalid credentials
	if strings.Contains(errStr, "UnrecognizedClientException") ||
		strings.Contains(errStr, "InvalidClientTokenId") ||
		strings.Contains(errStr, "SignatureDoesNotMatch") {
		authErr.Message = "AWS credentials are invalid or expired"
		authErr.Hint = buildInvalidCredentialsHint(profile)
		return authErr
	}

	// Check for SSO configuration errors
	if strings.Contains(errStr, "failed to load SSO") ||
		strings.Contains(errStr, "SSO") && strings.Contains(errStr, "not configured") {
		authErr.IsSSOErr = true
		authErr.Message = "AWS SSO configuration error"
		if profile != "" {
			authErr.Hint = fmt.Sprintf("Please check your AWS config for profile '%s' or run: aws sso login --profile %s",
				profile, profile)
		} else {
			authErr.Hint = "Please check your AWS SSO configuration in ~/.aws/config"
		}
		return authErr
	}

	// Generic authentication error
	authErr.Message = fmt.Sprintf("AWS authentication failed: %v", err)
	authErr.Hint = buildGenericAuthHint(profile)
	return authErr
}

func buildNoCredentialsHint(profile string) string {
	var hints []string
	hints = append(hints, "AWS credentials can be configured in several ways:")

	if profile != "" {
		hints = append(hints, fmt.Sprintf("  1. SSO: Run 'aws sso login --profile %s'", profile))
		hints = append(hints,
			fmt.Sprintf("  2. Configure profile in ~/.aws/config and ~/.aws/credentials for profile '%s'", profile))
	} else {
		hints = append(hints, "  1. SSO: Configure SSO in ~/.aws/config and run 'aws sso login'")
		hints = append(hints, "  2. Environment variables: AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
		hints = append(hints, "  3. Credentials file: ~/.aws/credentials (default profile)")
		hints = append(hints, "  4. IAM role: If running on EC2/ECS/Lambda")
	}

	return strings.Join(hints, "\n")
}

func buildInvalidCredentialsHint(profile string) string {
	var hints []string

	if profile != "" {
		hints = append(hints, fmt.Sprintf("If using SSO with profile '%s':", profile))
		hints = append(hints, fmt.Sprintf("  - Run: aws sso login --profile %s", profile))
		hints = append(hints, "")
		hints = append(hints, "If using access keys:")
		hints = append(hints, "  - Verify your AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
		hints = append(hints, fmt.Sprintf("  - Check credentials in ~/.aws/credentials for profile '%s'", profile))
	} else {
		hints = append(hints, "If using SSO:")
		hints = append(hints, "  - Run: aws sso login")
		hints = append(hints, "")
		hints = append(hints, "If using access keys:")
		hints = append(hints, "  - Verify your AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are correct")
		hints = append(hints, "  - Check credentials in ~/.aws/credentials")
	}

	return strings.Join(hints, "\n")
}

func buildGenericAuthHint(profile string) string {
	if profile != "" {
		return fmt.Sprintf("Try running 'aws sso login --profile %s' if using SSO, or verify your credentials configuration",
			profile)
	}
	return "Try running 'aws sso login' if using SSO, or verify your credentials configuration"
}

// DetectAuthMethod attempts to detect which authentication method is being used.
func DetectAuthMethod(profile string) string {
	// Check environment variables first
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		return "environment variables (AWS_ACCESS_KEY_ID)"
	}

	if os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI") != "" ||
		os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI") != "" {
		return "ECS task role"
	}

	// Check for web identity token (EKS)
	if os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE") != "" {
		return "web identity token (EKS/IRSA)"
	}

	// If profile is set, likely using shared config (could be SSO or credentials file)
	if profile != "" {
		return fmt.Sprintf("profile '%s' (SSO or credentials file)", profile)
	}

	return "default credentials chain"
}
