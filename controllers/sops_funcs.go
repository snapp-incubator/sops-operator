package controllers

import (
	"bytes"
	"fmt"
	"github.com/fatih/color"
	"github.com/goware/prefixer"
	"github.com/mitchellh/go-wordwrap"
	"go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/keys"
	"go.mozilla.org/sops/v3/keyservice"
	"go.mozilla.org/sops/v3/pgp"
	"go.mozilla.org/sops/v3/shamir"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

var statusSuccess = color.New(color.FgGreen).Sprint("SUCCESS")
var statusFailed = color.New(color.FgRed).Sprint("FAILED")

func GetDataKeyCustom(t sops.Metadata, passphrase string) ([]byte, error) {
	return GetDataKeyWithKeyServicesCustom([]keyservice.KeyServiceClient{
		keyservice.NewLocalClient(),
	}, t, passphrase)
}

func GetDataKeyWithKeyServicesCustom(svcs []keyservice.KeyServiceClient, m sops.Metadata, passphrase string) ([]byte, error) {
	getDataKeyErr := getDataKeyError{
		RequiredSuccessfulKeyGroups: m.ShamirThreshold,
		GroupResults:                make([]error, len(m.KeyGroups)),
	}
	var parts [][]byte
	for i, group := range m.KeyGroups {
		part, err := decryptKeyGroupCustom(group, svcs, passphrase)
		if err == nil {
			parts = append(parts, part)
		}
		getDataKeyErr.GroupResults[i] = err
	}
	var dataKey []byte
	if len(m.KeyGroups) > 1 {
		if len(parts) < m.ShamirThreshold {
			return nil, &getDataKeyErr
		}
		var err error
		dataKey, err = shamir.Combine(parts)
		if err != nil {
			return nil, fmt.Errorf("could not get data key from shamir parts: %s", err)
		}
	} else {
		if len(parts) != 1 {
			return nil, &getDataKeyErr
		}
		dataKey = parts[0]
	}
	return dataKey, nil
}

func decryptKeyGroupCustom(group sops.KeyGroup, svcs []keyservice.KeyServiceClient, passphrase string) ([]byte, error) {
	var keyErrs []error
	for _, key := range group {
		part, err := decryptKeyCustom(key, svcs, passphrase)
		if err != nil {
			keyErrs = append(keyErrs, err)
		} else {
			return part, nil
		}
	}
	return nil, decryptKeyErrors(keyErrs)
}

func decryptKeyCustom(key keys.MasterKey, svcs []keyservice.KeyServiceClient, passphrase string) ([]byte, error) {
	svcKey := keyservice.KeyFromMasterKey(key)
	var part []byte
	decryptErr := decryptKeyError{
		keyName: key.ToString(),
	}
	part, err := decryptWithPgp(svcKey.GetPgpKey().Fingerprint, key.EncryptedDataKey(), passphrase)
	if err != nil {
		return []byte{}, err
	}
	if part != nil {
		return part, nil
	}
	return nil, &decryptErr
}

func decryptWithPgp(fingerprint string, ciphertext []byte, passphrase string) ([]byte, error) {
	pgpKey := pgp.NewMasterKeyFromFingerprint(fingerprint)
	pgpKey.EncryptedKey = string(ciphertext)
	plaintext, err := DecryptWithGPGCustpm(pgpKey, passphrase)
	return []byte(plaintext), err
}

func DecryptWithGPGCustpm(key *pgp.MasterKey, passphrase string) ([]byte, error) {
	dataKey, binaryErr := decryptWithGPGBinaryCustom(key, passphrase)
	if binaryErr == nil {
		// log.WithField("fingerprint", key.Fingerprint).Info("Decryption succeeded")
		return dataKey, nil
	}
	// log.WithField("fingerprint", key.Fingerprint).Info("Decryption failed")
	return nil, fmt.Errorf(
		`could not decrypt data key with PGP key: GPG binary error: %v`, binaryErr)
}

func decryptWithGPGBinaryCustom(key *pgp.MasterKey, passphrase string) ([]byte, error) {
	args := []string{"--use-agent", "-d"}
	if passphrase != "" {
		args = append(args, []string{"--pinentry-mode=loopback", "--passphrase", passphrase}...)
	}
	cmd := exec.Command(gpgBinary(), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = strings.NewReader(key.EncryptedKey)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func gpgBinary() string {
	binary := "gpg"
	if envBinary := os.Getenv("SOPS_GPG_EXEC"); envBinary != "" {
		binary = envBinary
	}
	return binary
}

func NewMasterKeyFromFingerprint(fingerprint string) *pgp.MasterKey {
	return &pgp.MasterKey{
		Fingerprint:  strings.Replace(fingerprint, " ", "", -1),
		CreationDate: time.Now().UTC(),
	}
}

type decryptKeyErrors []error

func (e decryptKeyErrors) Error() string {
	return fmt.Sprintf("error decrypting key: %s", []error(e))
}

type getDataKeyError struct {
	RequiredSuccessfulKeyGroups int
	GroupResults                []error
}

func (err *getDataKeyError) successfulKeyGroups() int {
	n := 0
	for _, r := range err.GroupResults {
		if r == nil {
			n++
		}
	}
	return n
}

func (err *getDataKeyError) Error() string {
	return fmt.Sprintf("Error getting data key: %d successful groups "+
		"required, got %d", err.RequiredSuccessfulKeyGroups,
		err.successfulKeyGroups())
}

func (err *getDataKeyError) UserError() string {
	var groupErrs []string
	for i, res := range err.GroupResults {
		groupErr := decryptGroupError{
			err:       res,
			groupName: fmt.Sprintf("%d", i),
		}
		groupErrs = append(groupErrs, groupErr.UserError())
	}
	var trailer string
	if err.RequiredSuccessfulKeyGroups == 0 {
		trailer = "Recovery failed because no master key was able to decrypt " +
			"the file. In order for SOPS to recover the file, at least one key " +
			"has to be successful, but none were."
	} else {
		trailer = fmt.Sprintf("Recovery failed because the file was "+
			"encrypted with a Shamir threshold of %d, but only %d part(s) "+
			"were successfully recovered, one for each successful key group. "+
			"In order for SOPS to recover the file, at least %d groups have "+
			"to be successful. In order for a group to be successful, "+
			"decryption has to succeed with any of the keys in that key group.",
			err.RequiredSuccessfulKeyGroups, err.successfulKeyGroups(),
			err.RequiredSuccessfulKeyGroups)
	}
	trailer = wordwrap.WrapString(trailer, 75)
	return fmt.Sprintf("Failed to get the data key required to "+
		"decrypt the SOPS file.\n\n%s\n\n%s",
		strings.Join(groupErrs, "\n\n"), trailer)
}

type decryptGroupError struct {
	groupName string
	err       error
}

func (r *decryptGroupError) Error() string {
	return fmt.Sprintf("could not decrypt group %s: %s", r.groupName, r.err)
}

func (r *decryptGroupError) UserError() string {
	var status string
	if r.err == nil {
		status = statusSuccess
	} else {
		status = statusFailed
	}
	header := fmt.Sprintf(`Group %s: %s`, r.groupName, status)
	if r.err == nil {
		return header
	}
	message := r.err.Error()
	if userError, ok := r.err.(UserError); ok {
		message = userError.UserError()
	}
	reader := prefixer.New(strings.NewReader(message), "  ")
	// Safe to ignore this error, as reading from a strings.Reader can't fail
	errMsg, _ := ioutil.ReadAll(reader)
	return fmt.Sprintf("%s\n%s", header, string(errMsg))
}

type UserError interface {
	error
	UserError() string
}

type decryptKeyError struct {
	keyName string
	errs    []error
}

func (e *decryptKeyError) isSuccessful() bool {
	for _, err := range e.errs {
		if err == nil {
			return true
		}
	}
	return false
}

func (e *decryptKeyError) Error() string {
	return fmt.Sprintf("error decrypting key %s: %s", e.keyName, e.errs)
}

func (e *decryptKeyError) UserError() string {
	var status string
	if e.isSuccessful() {
		status = statusSuccess
	} else {
		status = statusFailed
	}
	header := fmt.Sprintf("%s: %s", e.keyName, status)
	if e.isSuccessful() {
		return header
	}
	var errMessages []string
	for _, err := range e.errs {
		wrappedErr := wordwrap.WrapString(err.Error(), 60)
		reader := prefixer.New(strings.NewReader(wrappedErr), "  | ")
		// Safe to ignore this error, as reading from a strings.Reader can't fail
		errMsg, _ := ioutil.ReadAll(reader)
		errMsg[0] = '-'
		errMessages = append(errMessages, string(errMsg))
	}
	joinedMsgs := strings.Join(errMessages, "\n\n")
	reader := prefixer.New(strings.NewReader(joinedMsgs), "  ")
	errMsg, _ := ioutil.ReadAll(reader)
	return fmt.Sprintf("%s\n%s", header, string(errMsg))
}
