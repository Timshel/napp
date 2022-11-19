package napp

import (
   "sort"
   "time"

   "github.com/superseriousbusiness/gotosocial/internal/log"
)

type ToCreate struct {
   ToPoll    *ToPoll
   Tweet     *NitterTweet
}


func (n *napper) refresh() {
   log.Infof("Initiate polling every %dminutes", n.pollFrequency)
   ticker := time.NewTicker(time.Duration(n.pollFrequency) * time.Minute)

   for {
      select {
         case <-n.ctx.Done(): return
         case <-ticker.C:
            var toCreate []ToCreate

            toPoll, err := n.GetAccountsToPoll(n.ctx)
            log.Infof("Started polling %d accounts", len(toPoll))
            if( err != nil ) {
               log.Errorf("Failed to retrieve accounts to poll: %s", err)
            }
            for _, infos := range toPoll {
               tweets, err := n.getNitterTimeline(FromIdDB(infos.DBAccountID))

               if( err != nil ){
                  log.Errorf("Failed to retrive tweets for account %s: %s", infos.DBAccountID, err)
               }
               for _, tweet := range tweets {
                  if( tweet.Time.After(infos.LastTweet) ){
                     toCreate = append(toCreate, ToCreate { ToPoll: infos, Tweet: tweet })
                  }
               }
            }

            sort.SliceStable(toCreate, func(i, j int) bool {
               return toCreate[i].Tweet.Time.Before(toCreate[j].Tweet.Time)
            })

            for _, create := range toCreate {
               err = n.PutStatus(n.ctx, create.ToPoll, create.Tweet)
               if( err != nil ) {
                  log.Errorf("Failed to create tweet %s: %s", create.Tweet, err)
               }
            }
      }
   }
}