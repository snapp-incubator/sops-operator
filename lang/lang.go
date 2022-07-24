package lang

// api variables
var (
	// ErrSopsSecretSpecGPGKeyRefNameEmpty when SopsSecret object's Spec.GPGKeyRefName is empty
	ErrSopsSecretSpecGPGKeyRefNameEmpty = "gpg_key_ref_name can't be empty in SopsSecret object"

	// ErrSopsSecretSpecSecretTemplateNameEmpty when SopsSecret object's Spec.SecretTemplate.Name is empty
	ErrSopsSecretSpecSecretTemplateNameEmpty = "name can't be empty in SopsSecret object"

	// ErrSopsSecretSpecNoData when SopsSecret object's Spec.SecretTemplate.Name is empty
	ErrSopsSecretSpecNoData = "data or stringData can't be empty in SopsSecret object"

	// ErrGPGKeySpecPassphraseLength when length of the provided password is not enough
	ErrGPGKeySpecPassphraseLength = "passphrase length is not long enough"

	// ErrGPGKeySpecArmoredPrivateKeyLength when length of the provided password is not enough
	ErrGPGKeySpecArmoredPrivateKeyLength = "armored key length can't be zero"
)

// controller variables
var (
	// ErrGPGKeyRefFetchFail when fails to fetch GPGKey object by name specified in SopsSecret.Spec.gpg_key_ref_name
	ErrGPGKeyRefFetchFail = "GPGKeyRefName error"
)
