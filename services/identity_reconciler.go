package services

import (
	"github.com/keratin/authn-server/config"
	"github.com/keratin/authn-server/data"
	"github.com/keratin/authn-server/lib"
	"github.com/keratin/authn-server/lib/oauth"
	"github.com/keratin/authn-server/models"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// IdentityReconciler will reconcile an OAuth identity with an AuthN account. This may mean:
//
// * finding the linked account
// * linking to an existing account
// * creating a new account
//
// Some expected errors include:
//
// * account is locked
// * linkable account is already linked
// * identity's email is already registered
func IdentityReconciler(accountStore data.AccountStore, cfg *config.Config, providerName string, providerUser *oauth.UserInfo, providerToken *oauth2.Token, linkableAccountID int) (*models.Account, error) {
	// 1. check for linked account
	linkedAccount, err := accountStore.FindByOauthAccount(providerName, providerUser.ID)
	if err != nil {
		return nil, errors.Wrap(err, "FindByOauthAccount")
	}
	if linkedAccount != nil {
		if linkedAccount.Locked {
			return nil, errors.New("account locked")
		}
		return linkedAccount, nil
	}

	// 2. attempt linking to existing account
	if linkableAccountID != 0 {
		err = accountStore.AddOauthAccount(linkableAccountID, providerName, providerUser.ID, providerToken.AccessToken)
		if err != nil {
			if data.IsUniquenessError(err) {
				return nil, errors.New("session conflict")
			}
			return nil, errors.Wrap(err, "AddOauthAccount")
		}
		sessionAccount, err := accountStore.Find(linkableAccountID)
		if err != nil {
			return nil, errors.Wrap(err, "Find")
		}
		return sessionAccount, nil
	}

	// 3. attempt creating new account
	rand, err := lib.GenerateToken()
	if err != nil {
		return nil, errors.Wrap(err, "GenerateToken")
	}
	// TODO: transactional account + identity
	newAccount, err := AccountCreator(accountStore, cfg, providerUser.Email, string(rand))
	if err != nil {
		return nil, errors.Wrap(err, "AccountCreator")
	}
	accountStore.AddOauthAccount(newAccount.ID, providerName, providerUser.ID, providerToken.AccessToken)
	return newAccount, nil
}
