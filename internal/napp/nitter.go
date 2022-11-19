package napp

import (
   "encoding/json"
   "fmt"
   "net/http"
   "strconv"
   "strings"
   "time"

   "codeberg.org/gruf/go-kv"
   "github.com/superseriousbusiness/gotosocial/internal/log"
)

type NitterUser struct {
    Id                    string `json:"id"`
    Handle                string `json:"username"`
    DisplayName           string `json:"fullname"`
    AvatarURL             string `json:"userPic"`
    BackgoundURL          string `json:"banner"`
}

func (n *napper) getJson(url string, target interface{}) error {
    request, err := http.NewRequest(http.MethodGet, url, nil)
    if err != nil {
        return err
    }

    r, err := n.httpclient.Do(request)
    if err != nil {
        return err
    }
    defer r.Body.Close()
    return json.NewDecoder(r.Body).Decode(target)
}

func (n*napper) getNitterUser(username string) (*NitterUser, error) {
    user := NitterUser{}
    url := fmt.Sprintf("%s/api/user/%s", n.nitterHost, username)
    err := n.getJson(url, &user)

    if( err == nil ){
        user.AvatarURL = "https://" + user.AvatarURL
        if( strings.HasPrefix(user.BackgoundURL, "#") ){
            user.BackgoundURL = ""
        }
    }

    return &user, err
}

type NitterCard struct{
    Url          string
    Image        string
}

type NitterTweet struct{
    Id          int64
    ThreadId    int64
    ReplyId     int64
    Reply       []string
    Time        time.Time
    Text        string
    Card        *NitterCard
    User        *NitterUser
    Photos      []string
    NitterURL   string
}

func (n*napper) getNitterTimeline(accountId string) ([]*NitterTweet, error) {
    l := log.WithFields(kv.Fields{ { K: "ID", V: accountId,}, }...)

    tweets := make([]*NitterTweet, 0)

    url := fmt.Sprintf("%s/api/timeline/%s", n.nitterHost, accountId)

    err := n.getJson(url, &tweets)

    for _, tweet := range tweets {
        tweet.NitterURL = fmt.Sprintf("%s/%s/status/%s", 
            n.nitterHost, tweet.User.Handle, strconv.FormatInt(tweet.ThreadId, 10))

        if( tweet.Card != nil ){
            tweet.Card.Image = "https://" + tweet.Card.Image
        }

        for i, photo := range tweet.Photos {
            tweet.Photos[i] = fmt.Sprintf("%s/pic/orig/%s", n.nitterHost, photo)
        }
    }

    if( err != nil ){
        l.Errorf("Failed to retrieve Timeline: %s", err)
    }

    return tweets, err
}