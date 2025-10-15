package awsclient

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthError_Error(t *testing.T) {
	tests := []struct {
		name     string
		authErr  *AuthError
		expected string
	}{
		{
			name: "message only",
			authErr: &AuthError{
				Message: "Test error",
			},
			expected: "Test error",
		},
		{
			name: "message with hint",
			authErr: &AuthError{
				Message: "Test error",
				Hint:    "Try this fix",
			},
			expected: "Test error\n\nTry this fix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.authErr.Error())
		})
	}
}

func TestAuthError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	authErr := &AuthError{
		Original: originalErr,
		Message:  "wrapped",
	}

	assert.Equal(t, originalErr, authErr.Unwrap())
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		isAuthErr bool
	}{
		{
			name:      "nil error",
			err:       nil,
			isAuthErr: false,
		},
		{
			name:      "regular error",
			err:       errors.New("regular error"),
			isAuthErr: false,
		},
		{
			name: "auth error",
			err: &AuthError{
				Message: "auth failed",
			},
			isAuthErr: true,
		},
		{
			name:      "wrapped auth error",
			err:       wrapAuthError(errors.New("failed to refresh cached credentials"), "test-profile"),
			isAuthErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authErr, ok := IsAuthError(tt.err)
			assert.Equal(t, tt.isAuthErr, ok)
			if ok {
				assert.NotNil(t, authErr)
			} else {
				assert.Nil(t, authErr)
			}
		})
	}
}

func TestWrapAuthError_SSOTokenExpiration(t *testing.T) {
	tests := []struct {
		name             string
		errMsg           string
		profile          string
		expectedMsg      string
		expectedHint     string
		expectedIsSSOErr bool
	}{
		{
			name:             "SSO token expired with profile",
			errMsg:           "failed to refresh cached credentials",
			profile:          "my-sso-profile",
			expectedMsg:      "AWS SSO session has expired or is invalid",
			expectedHint:     "Please run: aws sso login --profile my-sso-profile",
			expectedIsSSOErr: true,
		},
		{
			name:             "SSO session expired",
			errMsg:           "the SSO session has expired",
			profile:          "",
			expectedMsg:      "AWS SSO session has expired or is invalid",
			expectedHint:     "Please run: aws sso login",
			expectedIsSSOErr: true,
		},
		{
			name:             "SSO session invalid",
			errMsg:           "SSO session is invalid",
			profile:          "test",
			expectedMsg:      "AWS SSO session has expired or is invalid",
			expectedHint:     "Please run: aws sso login --profile test",
			expectedIsSSOErr: true,
		},
		{
			name:             "token is expired",
			errMsg:           "token is expired",
			profile:          "",
			expectedMsg:      "AWS SSO session has expired or is invalid",
			expectedHint:     "Please run: aws sso login",
			expectedIsSSOErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wrapAuthError(errors.New(tt.errMsg), tt.profile)
			require.Error(t, err)

			authErr, ok := IsAuthError(err)
			require.True(t, ok)
			assert.Equal(t, tt.expectedMsg, authErr.Message)
			assert.Equal(t, tt.expectedHint, authErr.Hint)
			assert.Equal(t, tt.expectedIsSSOErr, authErr.IsSSOErr)
			assert.Equal(t, tt.profile, authErr.Profile)
		})
	}
}

func TestWrapAuthError_SSOWithEnvProfile(t *testing.T) {
	// Set AWS_PROFILE env var
	originalProfile := os.Getenv("AWS_PROFILE")
	_ = os.Setenv("AWS_PROFILE", "env-profile")
	defer func() {
		if originalProfile != "" {
			_ = os.Setenv("AWS_PROFILE", originalProfile)
		} else {
			_ = os.Unsetenv("AWS_PROFILE")
		}
	}()

	err := wrapAuthError(errors.New("failed to refresh cached credentials"), "")
	require.Error(t, err)

	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Contains(t, authErr.Hint, "aws sso login --profile env-profile")
}

func TestWrapAuthError_NoCredentials(t *testing.T) {
	tests := []struct {
		name        string
		errMsg      string
		profile     string
		expectedMsg string
	}{
		{
			name:        "no valid credentials",
			errMsg:      "no valid credentials found",
			profile:     "",
			expectedMsg: "No AWS credentials found",
		},
		{
			name:        "NoCredentialProviders",
			errMsg:      "NoCredentialProviders: no valid providers in chain",
			profile:     "test",
			expectedMsg: "No AWS credentials found",
		},
		{
			name:        "could not find credentials",
			errMsg:      "could not find credentials",
			profile:     "",
			expectedMsg: "No AWS credentials found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wrapAuthError(errors.New(tt.errMsg), tt.profile)
			require.Error(t, err)

			authErr, ok := IsAuthError(err)
			require.True(t, ok)
			assert.Equal(t, tt.expectedMsg, authErr.Message)
			assert.NotEmpty(t, authErr.Hint)
			assert.Contains(t, authErr.Hint, "AWS credentials can be configured")
		})
	}
}

func TestWrapAuthError_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name        string
		errMsg      string
		profile     string
		expectedMsg string
	}{
		{
			name:        "UnrecognizedClientException",
			errMsg:      "UnrecognizedClientException: The security token included in the request is invalid",
			profile:     "my-profile",
			expectedMsg: "AWS credentials are invalid or expired",
		},
		{
			name:        "InvalidClientTokenId",
			errMsg:      "InvalidClientTokenId: The security token is invalid",
			profile:     "",
			expectedMsg: "AWS credentials are invalid or expired",
		},
		{
			name:        "SignatureDoesNotMatch",
			errMsg:      "SignatureDoesNotMatch: The request signature we calculated does not match",
			profile:     "test",
			expectedMsg: "AWS credentials are invalid or expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wrapAuthError(errors.New(tt.errMsg), tt.profile)
			require.Error(t, err)

			authErr, ok := IsAuthError(err)
			require.True(t, ok)
			assert.Equal(t, tt.expectedMsg, authErr.Message)
			assert.NotEmpty(t, authErr.Hint)
		})
	}
}

func TestWrapAuthError_SSOConfigError(t *testing.T) {
	tests := []struct {
		name             string
		errMsg           string
		profile          string
		expectedMsg      string
		expectedIsSSOErr bool
	}{
		{
			name:             "failed to load SSO",
			errMsg:           "failed to load SSO credentials",
			profile:          "sso-profile",
			expectedMsg:      "AWS SSO configuration error",
			expectedIsSSOErr: true,
		},
		{
			name:             "SSO not configured",
			errMsg:           "SSO not configured for profile",
			profile:          "",
			expectedMsg:      "AWS SSO configuration error",
			expectedIsSSOErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wrapAuthError(errors.New(tt.errMsg), tt.profile)
			require.Error(t, err)

			authErr, ok := IsAuthError(err)
			require.True(t, ok)
			assert.Equal(t, tt.expectedMsg, authErr.Message)
			assert.Equal(t, tt.expectedIsSSOErr, authErr.IsSSOErr)
			assert.NotEmpty(t, authErr.Hint)
		})
	}
}

func TestWrapAuthError_GenericError(t *testing.T) {
	err := wrapAuthError(errors.New("some other AWS error"), "test-profile")
	require.Error(t, err)

	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Contains(t, authErr.Message, "AWS authentication failed")
	assert.NotEmpty(t, authErr.Hint)
}

func TestDetectAuthMethod(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		cleanup  func()
		profile  string
		expected string
	}{
		{
			name: "AWS_ACCESS_KEY_ID env var",
			setup: func() {
				_ = os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
			},
			cleanup: func() {
				_ = os.Unsetenv("AWS_ACCESS_KEY_ID")
			},
			profile:  "",
			expected: "environment variables (AWS_ACCESS_KEY_ID)",
		},
		{
			name: "ECS task role",
			setup: func() {
				_ = os.Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "/v2/credentials/xxx")
			},
			cleanup: func() {
				_ = os.Unsetenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")
			},
			profile:  "",
			expected: "ECS task role",
		},
		{
			name: "Web identity token (EKS)",
			setup: func() {
				_ = os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "/var/run/secrets/eks.amazonaws.com/serviceaccount/token")
			},
			cleanup: func() {
				_ = os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
			},
			profile:  "",
			expected: "web identity token (EKS/IRSA)",
		},
		{
			name:     "Profile specified",
			setup:    func() {},
			cleanup:  func() {},
			profile:  "my-profile",
			expected: "profile 'my-profile' (SSO or credentials file)",
		},
		{
			name:     "Default credentials chain",
			setup:    func() {},
			cleanup:  func() {},
			profile:  "",
			expected: "default credentials chain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			result := DetectAuthMethod(tt.profile)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildNoCredentialsHint(t *testing.T) {
	tests := []struct {
		name             string
		profile          string
		expectedContains []string
	}{
		{
			name:    "with profile",
			profile: "my-profile",
			expectedContains: []string{
				"AWS credentials can be configured",
				"aws sso login --profile my-profile",
				"~/.aws/config",
			},
		},
		{
			name:    "without profile",
			profile: "",
			expectedContains: []string{
				"AWS credentials can be configured",
				"Environment variables: AWS_ACCESS_KEY_ID",
				"~/.aws/credentials",
				"IAM role: If running on EC2/ECS/Lambda",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := buildNoCredentialsHint(tt.profile)
			for _, expected := range tt.expectedContains {
				assert.Contains(t, hint, expected)
			}
		})
	}
}

func TestBuildInvalidCredentialsHint(t *testing.T) {
	tests := []struct {
		name             string
		profile          string
		expectedContains []string
	}{
		{
			name:    "with profile",
			profile: "my-profile",
			expectedContains: []string{
				"If using SSO",
				"aws sso login --profile my-profile",
				"If using access keys",
				"AWS_ACCESS_KEY_ID",
			},
		},
		{
			name:    "without profile",
			profile: "",
			expectedContains: []string{
				"If using SSO",
				"aws sso login",
				"If using access keys",
				"~/.aws/credentials",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := buildInvalidCredentialsHint(tt.profile)
			for _, expected := range tt.expectedContains {
				assert.Contains(t, hint, expected)
			}
		})
	}
}

func TestBuildGenericAuthHint(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		expected string
	}{
		{
			name:     "with profile",
			profile:  "my-profile",
			expected: "Try running 'aws sso login --profile my-profile' if using SSO, or verify your credentials configuration",
		},
		{
			name:     "without profile",
			profile:  "",
			expected: "Try running 'aws sso login' if using SSO, or verify your credentials configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildGenericAuthHint(tt.profile)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateCredentials(t *testing.T) {
	ctx := context.Background()

	t.Run("valid credentials", func(t *testing.T) {
		// This test would need proper AWS credentials or mocking
		// For now, we test the error handling path
		cfg := aws.Config{}

		err := ValidateCredentials(ctx, cfg, "test-profile")
		// We expect an error since we're using an empty config
		assert.Error(t, err)

		// Verify it returns an AuthError
		authErr, ok := IsAuthError(err)
		assert.True(t, ok)
		assert.NotNil(t, authErr)
	})
}

func TestWrapAuthError_NilError(t *testing.T) {
	err := wrapAuthError(nil, "test-profile")
	assert.Nil(t, err)
}

func TestAuthError_HintFormatting(t *testing.T) {
	// Test that multi-line hints are properly formatted
	err := wrapAuthError(errors.New("no valid credentials"), "test-profile")
	require.Error(t, err)

	authErr, ok := IsAuthError(err)
	require.True(t, ok)

	errorString := authErr.Error()
	assert.Contains(t, errorString, "No AWS credentials found")
	assert.Contains(t, errorString, "\n\n")               // Check for proper spacing
	assert.True(t, strings.Count(errorString, "\n") >= 2) // Multiple lines in hint
}
