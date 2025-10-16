package awsclient

// NewIdentityForTest creates an Identity for testing purposes
func NewIdentityForTest(account, arn, userID string) Identity {
	return identity{
		account: account,
		arn:     arn,
		userID:  userID,
	}
}
