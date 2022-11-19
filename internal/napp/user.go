package napp

import (
   "context"
   "crypto/rsa"
   "crypto/rand"
   "fmt"
   "time"

   "codeberg.org/gruf/go-kv"
   "github.com/superseriousbusiness/gotosocial/internal/ap"
   "github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
   "github.com/superseriousbusiness/gotosocial/internal/log"
   "github.com/superseriousbusiness/gotosocial/internal/uris"
   "golang.org/x/crypto/bcrypt"
)

func (n *napper) NewUser(ctx context.Context, username string) error {
   l := log.WithFields(kv.Fields{{K: "username", V: username},}...)
   dbUsername := TolUsernameDB(username)

   key, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
   if err != nil {
      l.Errorf("error creating new rsa key: %s", err)
      return err
   }

   // if something went wrong while creating a user, we might already have an account, so check here first...
   available, err := n.db.IsUsernameAvailable(ctx, dbUsername)
   if available && err == nil {

      nu, err := n.getNitterUser(username)
      if err != nil {
         return fmt.Errorf("Failed to fetch nitter user: %s", err)
      }

      // if we have db.ErrNoEntries, we just don't have an
      // account yet so create one before we proceed
      accountURIs := uris.GenerateURIsForAccount(dbUsername)

      userId := ToIdDB(nu.Id)
      twitterUrl := gtsmodel.Field { 
         Name: "Twitter",
         Value: fmt.Sprintf("https://twitter.com/%s", nu.Handle),
      }
      acct := &gtsmodel.Account{
         ID:                    userId,
         Username:              dbUsername,
         DisplayName:           fmt.Sprintf("🤖 %s 🤖", nu.DisplayName),
         Note:                  "I'm a bot duplicating content from Twitter",
         Bot:                   &[]bool{true}[0],
         Locked:                &[]bool{false}[0],
         Discoverable:          &[]bool{false}[0],
         Privacy:               gtsmodel.VisibilityFollowersOnly,
         URL:                   n.nitterHost + "/" + dbUsername,
         PrivateKey:            key,
         PublicKey:             &key.PublicKey,
         PublicKeyURI:          accountURIs.PublicKeyURI,
         ActorType:             ap.ActorPerson,
         URI:                   accountURIs.UserURI,
         AvatarRemoteURL:       nu.AvatarURL,
         HeaderRemoteURL:       nu.BackgoundURL,
         InboxURI:              accountURIs.InboxURI,
         OutboxURI:             accountURIs.OutboxURI,
         FollowersURI:          accountURIs.FollowersURI,
         FollowingURI:          accountURIs.FollowingURI,
         FeaturedCollectionURI: accountURIs.CollectionURI,
         Fields:                []gtsmodel.Field{ twitterUrl },
      }

      _, err = n.dereferencer.FetchRemoteAccountMedia(ctx, acct, "admin", true)
      if err != nil {
         return fmt.Errorf("Error fetching account (%s) media: %s", dbUsername, err)
      }

      // insert the new account!
      if err := n.db.PutAccount(ctx, acct); err != nil {
         return err
      }

      pw, err := bcrypt.GenerateFromPassword([]byte(n.userPassword), bcrypt.DefaultCost)
      if err != nil {
         return fmt.Errorf("error hashing password: %s", err)
      }

      u := &gtsmodel.User{
         ID:                     acct.ID,
         AccountID:              acct.ID,
         Account:                acct,
         EncryptedPassword:      string(pw),
         Email:                  nu.Handle + "@twitter.com",
         ConfirmedAt:            time.Now(),
         Approved:               &[]bool{true}[0],
      }

      // insert the user!
      err = n.db.PutUser(ctx, u)
   }

   return err;
}