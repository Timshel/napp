package napp

import (
   "context"
   "errors"
   "fmt"

   "codeberg.org/gruf/go-runners"
   "github.com/superseriousbusiness/gotosocial/internal/config"
   "github.com/superseriousbusiness/gotosocial/internal/db"
   "github.com/superseriousbusiness/gotosocial/internal/db/bundb"
   "github.com/superseriousbusiness/gotosocial/internal/federation/dereferencing"
   "github.com/superseriousbusiness/gotosocial/internal/httpclient"
   "github.com/superseriousbusiness/gotosocial/internal/media"
   "github.com/superseriousbusiness/gotosocial/internal/messages"
   "github.com/superseriousbusiness/gotosocial/internal/transport"
   "github.com/superseriousbusiness/gotosocial/internal/typeutils"
)

// generate RSA keys of this length
const rsaKeyBits = 2048

// Logic to proxy a Nitter instance
type Napper interface {

   // Start starts the Napper, start fetching Status from Nitter
   Start() error

   // Stop stops the Napper cleanly
   Stop() error

   NewUser(ctx context.Context, username string) error
}

// napper just implements the Napper interface
type napper struct {
   cw             *runners.WorkerPool[messages.FromClientAPI]
   db             db.DB
   dbConn         *bundb.DBConn
   dereferencer   dereferencing.Dereferencer
   httpclient     *httpclient.Client
   ctx            context.Context
   cancelFunc     context.CancelFunc

   nitterHost     string
   userPassword   string
   pollFrequency  int
}

// NewNapper returns a new Napper.
func NewNapper(
   pCtx                 context.Context,
   clientWorker         *runners.WorkerPool[messages.FromClientAPI],
   db                   db.DB, 
   dbConn               *bundb.DBConn,
   mediaManager         media.Manager,
   transportController  transport.Controller, 
   typeConverter        typeutils.TypeConverter,
) Napper {
   ctx, cancelFunc := context.WithCancel(pCtx)

   return &napper{
      cw:            clientWorker,
      db:            db,
      dbConn:        dbConn,
      dereferencer:  dereferencing.NewDereferencer(db, typeConverter, transportController, mediaManager),
      httpclient:    httpclient.New(httpclient.Config{ MaxOpenConnsPerHost: 1, }),
      ctx:           ctx,
      cancelFunc:    cancelFunc,
      nitterHost:    config.GetNappNitterHost(),
      userPassword:  config.GetNappUserPassword(),
      pollFrequency: config.GetNappPollFrequency(),
   }
}

// Start starts the Napper, start fetching Status from Nitter
func (n *napper) Start() error {

   if( n.nitterHost == "" ){
      return errors.New(fmt.Sprintf("Missing %s config", config.NappNitterHostFlag()))
   }

   if( n.userPassword == "" ){
      return errors.New(fmt.Sprintf("Missing %s config", config.NappUserPasswordFlag()))
   }

   if( n.pollFrequency == 0 ){
      return errors.New(fmt.Sprintf("Missing or invalid %s config %s", config.NappPollFrequencyFlag(), n.pollFrequency))
   }

   go n.refresh()
   return nil
}

// Stop stops the Napper cleanly
func (n *napper) Stop() error {
   n.cancelFunc()
   return nil
}
