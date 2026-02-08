package vault

// Event topics published by the Vault module.
const (
	TopicVaultStatusChanged = "vault.status.changed"
	TopicCredentialCreated  = "vault.credential.created"  //nolint:gosec // G101: event topic name, not a credential
	TopicCredentialUpdated  = "vault.credential.updated"  //nolint:gosec // G101: event topic name, not a credential
	TopicCredentialDeleted  = "vault.credential.deleted"  //nolint:gosec // G101: event topic name, not a credential
	TopicKeysRotated        = "vault.keys.rotated"
)
