// Package github provides you access to Github's OAuth2
// infrastructure.
package github

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	oauth2gh "golang.org/x/oauth2/github"
)

// Credentials stores google client-ids.
type Credentials struct {
	ClientID     string `json:"clientid"`
	ClientSecret string `json:"secret"`
}

// AuthUser rapresents datas of authenticated user
type AuthUser struct {
	Login              string `json:"login"`
	Name               string `json:"name"`
	OrganizationNeeded bool   `json:"organization_needed"`
}

var (
	conf                      *oauth2.Config
	cred                      Credentials
	state                     string
	store                     sessions.CookieStore
	authenticationRedirectURL string
	organizationRequired      string
)

// Session is the function needed by the handler github to initialize the session
func Session(name string) gin.HandlerFunc {
	return sessions.Sessions(name, store)
}

// Setup setup the github oauth2 handler
func Setup(credFile string, scopes []string, secret []byte, authRedirectURL, orgRequired string) {
	store = sessions.NewCookieStore(secret)
	var c Credentials
	file, err := ioutil.ReadFile(credFile)
	if err != nil {
		glog.Fatalf("[Gin-OAuth] File error: %v\n", err)
	}
	json.Unmarshal(file, &c)
	conf = &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Scopes:       scopes,
		Endpoint:     oauth2gh.Endpoint,
	}
	authenticationRedirectURL = authRedirectURL
	organizationRequired = orgRequired
}

// LoginHandler save in the cookie the state and return the url needed for authentication with Github
func LoginHandler(ctx *gin.Context) {
	state = randToken()
	thisSession := sessions.Default(ctx)
	thisSession.Set("state", state)
	thisSession.Save()

	response := struct {
		GithubURI string `json:"github_uri"`
	}{getLoginURL(state)}

	log.Printf("Returning URL: %+v", response)
	ctx.JSON(http.StatusOK, response)
}

// Auth initialize the authentication with Github
func Auth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Handle the exchange code to initiate a transport.
		thisSession := sessions.Default(ctx)
		retrievedState := thisSession.Get("state")
		log.Printf("retrievedState is %v", retrievedState)
		if retrievedState != ctx.Query("state") {
			ctx.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session state: %s", retrievedState))
			return
		}

		tok, err := conf.Exchange(oauth2.NoContext, ctx.Query("code"))
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}
		log.Printf("storing access_token: %v", tok)
		thisSession.Set("access_token", tok.AccessToken)
		thisSession.Set("refresh_token", tok.RefreshToken)
		thisSession.Save()
		tokRead := thisSession.Get("access_token")
		log.Printf("access_token read: %v", tokRead)
		log.Printf("session_auth is %+v", thisSession)
		ctx.Redirect(http.StatusMovedPermanently, authenticationRedirectURL)
		log.Printf("access_token2 read2: %v", tokRead)
	}
}

// CheckAuthenticatedUser will be implemented as a controller for authentication for every api in a group
func CheckAuthenticatedUser() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		thisSession := sessions.Default(ctx)
		log.Printf("session is %+v", thisSession)
		tok := thisSession.Get("access_token")
		log.Printf("access_token read: %v", tok)

		accessToken, ok := tok.(string)
		if !ok {
			ctx.AbortWithError(http.StatusUnauthorized, fmt.Errorf("missing accessToken: %T = %+v", tok, tok))
			return
		}
		myToken := oauth2.Token{
			AccessToken: accessToken,
		}
		oauthClient := conf.Client(oauth2.NoContext, &myToken)
		fmt.Printf("oauthClient is %+v \n", oauthClient)
		client := github.NewClient(oauthClient)
		fmt.Printf("client  github is %+v \n", client)
		fmt.Printf("client_github_user is %+v \n", client.Users)
		user, _, err := client.Users.Get(oauth2.NoContext, "")
		fmt.Printf("user get from github is %v \n", user)
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("error getting user: %v", err))
			return
		}

		isMember, resp, err := client.Organizations.IsMember(ctx, organizationRequired, user.GetLogin())
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("error getting user membership: %v, %v", err, resp))
			return
		}
		if isMember {
			fmt.Printf("the user %v is part of the membership required", user.GetLogin())
			setUser := AuthUser{
				Login:              user.GetLogin(),
				Name:               user.GetName(),
				OrganizationNeeded: isMember,
			}
			ctx.Set("authenticated_user", setUser)
			thisSession.Set("authenticated_user", setUser)
			thisSession.Save()
			ctx.Next()

		} else {
			fmt.Printf("the user %v is not part of the membership required", user.GetLogin())
			ctx.AbortWithError(http.StatusForbidden, fmt.Errorf("the user does not belong to the write organization"))
		}
	}
}

func randToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func getLoginURL(state string) string {
	return conf.AuthCodeURL(state)
}
