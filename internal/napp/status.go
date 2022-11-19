package napp

import (
	"context"
	"fmt"
	"time"
	"strings"

	"codeberg.org/gruf/go-kv"
	"github.com/superseriousbusiness/gotosocial/internal/ap"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/messages"
	"github.com/superseriousbusiness/gotosocial/internal/log"
	"github.com/superseriousbusiness/gotosocial/internal/uris"
)

func (n *napper) FormatContent(tweet *NitterTweet) string {
	urlPreview := strings.TrimLeft(tweet.NitterURL, "https://")
	return fmt.Sprintf("<p>%s<br>Source: <a href=\"%s\">%.36s...</a></p>", tweet.Text, tweet.NitterURL, urlPreview)
}


func (n *napper) InReply(accountURIs *uris.UserURIs, tweet *NitterTweet) (string, string, string) {
	if( tweet.ReplyId != 0 && len(tweet.Reply) == 1 && tweet.Reply[0] == tweet.User.Handle ){
		replyId := ToIdDB(tweet.ReplyId)
		replyURI := accountURIs.StatusesURI + "/" + replyId
		replyAccountId := fmt.Sprintf("%026s", tweet.User.Id)

		return replyId, replyURI, replyAccountId
	}
	return "", "", ""
}

func (n *napper) PutStatus(ctx context.Context, toPoll *ToPoll, tweet *NitterTweet) error {
	l := log.WithFields(kv.Fields{
		{ K: "ID", V: toPoll.DBAccountID,},
		{ K: "tweetId", V: tweet.Id,},
	}...)

	accountURIs := uris.GenerateURIsForAccount(toPoll.DBUsername)
	tweetId := ToIdDB(tweet.Id)

	inrId, inrURL, inrAId := n.InReply(accountURIs, tweet)

	attachments := make([]*gtsmodel.MediaAttachment, len(tweet.Photos))
  for i, photo := range tweet.Photos {
      attch := gtsmodel.MediaAttachment{ RemoteURL: photo, }
      attachments[i] = &attch
  }
	newStatus := &gtsmodel.Status{
		ID:                       tweetId,
		URI:                      accountURIs.StatusesURI + "/" + tweetId,
		URL:                      tweet.NitterURL,
		Content: 									n.FormatContent(tweet),
		Attachments: 							attachments,
		CreatedAt:                tweet.Time,
		UpdatedAt:                time.Now(),
		Local:                    &[]bool{true}[0],
		AccountID:                toPoll.DBAccountID,
		AccountURI:               accountURIs.UserURI,
		InReplyToID:              inrId,
		InReplyToURI:             inrURL,
		InReplyToAccountID: 			inrAId,
		ActivityStreamsType:      ap.ObjectNote,
		Visibility: 							gtsmodel.VisibilityFollowersOnly,
		Sensitive:                &[]bool{false}[0],
		Text:                     tweet.Text,
		Federated: 								&[]bool{true}[0],
		Boostable: 								&[]bool{false}[0],
		Replyable: 								&[]bool{false}[0],
		Likeable: 								&[]bool{true}[0],
	}

	n.dereferencer.PopulateStatusAttachments(n.ctx, newStatus, "admin")

	// put the new status in the database
	l.Infof(fmt.Sprintf("Pushing tweet to DB (time: %s)", tweet.Time))
	if err := n.db.PutStatus(ctx, newStatus); err != nil {
		l.Errorf("Failed to push tweet to DB: %s", err)
		return gtserror.NewErrorInternalError(err)
	}

	// send it back to the processor for async processing
	n.cw.Queue(messages.FromClientAPI{
		APObjectType:   ap.ObjectNote,
		APActivityType: ap.ActivityCreate,
		GTSModel:       newStatus,
		OriginAccount:  &gtsmodel.Account{
       ID:                    toPoll.DBAccountID,
       Username:              toPoll.DBUsername,
       PublicKeyURI:          accountURIs.PublicKeyURI,
       URI:                   accountURIs.UserURI,
       InboxURI:              accountURIs.InboxURI,
       OutboxURI:             accountURIs.OutboxURI,
       FollowersURI:          accountURIs.FollowersURI,
       FollowingURI:          accountURIs.FollowingURI,
       FeaturedCollectionURI: accountURIs.CollectionURI,
     },
	})

	return nil
}
