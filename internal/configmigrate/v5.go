package configmigrate

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// migrateTo5 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 4
//	'auth_name': …
//	'auth_pass': …
//	# …
//
//	# AFTER:
//	'schema_version': 5
//	'users':
//	- 'name': …
//	  'password': <hashed>
//	# …
func (m *Migrator) migrateTo5(_ context.Context, diskConf yobj) (err error) {
	diskConf["schema_version"] = 5

	user := yobj{}

	if err = moveVal[string](diskConf, user, "auth_name", "name"); err != nil {
		return err
	}

	pass, ok, err := fieldVal[string](diskConf, "auth_pass")
	if !ok {
		return err
	}
	delete(diskConf, "auth_pass")

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generating password hash: %w", err)
	}

	user["password"] = string(hash)
	diskConf["users"] = yarr{user}

	return nil
}
