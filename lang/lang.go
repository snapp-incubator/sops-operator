package lang

// api variables
var (
	// ErrSopsSecretSpecGPGKeyRefNameEmpty when SopsSecret object's Spec.GPGKeyRefName is empty
	ErrSopsSecretSpecGPGKeyRefNameEmpty = "gpg_key_ref_name can't be empty in SopsSecret object"

	// ErrSopsSecretSpecNoData when SopsSecret object's Spec.SecretTemplate.Name is empty
	ErrSopsSecretSpecNoData = "stringData can't be empty in SopsSecret object"

	// ErrGPGKeySpecPassphraseLength when length of the provided password is not enough
	ErrGPGKeySpecPassphraseLength = "passphrase length should be greater equal to 14 and lower equal to 100"

	// ErrGPGKeySpecPassphraseCommon when password is widely used and considered a bad password
	ErrGPGKeySpecPassphraseCommon = "passphrase is so common, please use another more complex password"

	// ErrGPGKeySpecArmoredPrivateKeyLength when length of the provided password is not enough
	ErrGPGKeySpecArmoredPrivateKeyLength = "armored key length can't be empty"

	// ErrGPGKeySpecArmoredPrivateKeyPrefixSuffix when key string has the pgp key prefix or suffix
	ErrGPGKeySpecArmoredPrivateKeyPrefixSuffix = "object GPGKey on field ArmoredPrivateKey should not have prefix or suffix of dashes"
)

// controller variables
var (
	// ErrGPGKeyRefFetchFail when fails to fetch GPGKey object by name specified in SopsSecret.Spec.gpg_key_ref_name
	ErrGPGKeyRefFetchFail = "GPGKeyRefName error"
)
