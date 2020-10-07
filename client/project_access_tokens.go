package client

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
)

// ProjectAccessToken represents a Rollbar project access token.
type ProjectAccessToken struct {
	Name                 string `fake:"{hackernoun}"`
	ProjectID            int    `json:"project_id" fake:"{number1,1000000}"`
	AccessToken          string `json:"access_token"`
	Scopes               []ProjectAccessTokenScope
	Status               Status
	RateLimitWindowSize  int `json:"rate_limit_window_size"`
	RateLimitWindowCount int `json:"rate_limit_window_count"`
	DateCreated          int `json:"date_created"`
	DateModified         int `json:"date_modified"`
}

// ListProjectAccessTokens lists the Rollbar project access tokens for the
// specified Rollbar project.
func (c *RollbarApiClient) ListProjectAccessTokens(projectID int) ([]ProjectAccessToken, error) {
	u := apiUrl + pathPatList

	l := log.With().
		Str("url", u).
		Logger()

	resp, err := c.resty.R().
		SetResult(patListResponse{}).
		SetError(ErrorResult{}).
		SetPathParams(map[string]string{
			"projectId": strconv.Itoa(projectID),
		}).
		Get(u)
	if err != nil {
		l.Err(err).Send()
		return nil, err
	}
	switch resp.StatusCode() {
	case http.StatusOK:
		pats := resp.Result().(*patListResponse).Result
		return pats, nil
	case http.StatusNotFound:
		l.Warn().Msg("Project not found")
		return nil, ErrNotFound
	case http.StatusUnauthorized:
		l.Warn().Msg("Unauthorized")
		return nil, ErrUnauthorized
	default:
		errResp := resp.Error().(*ErrorResult)
		l.Err(errResp).Msg("Unexpected error")
		return nil, errResp
	}
}

// ReadProjectAccessToken reads a Rollbar project access token from the API.  It
// returns the first token that matches `name`. If no matching token is found,
// returns error ErrNotFound.
func (c *RollbarApiClient) ReadProjectAccessToken(projectID int, name string) (ProjectAccessToken, error) {
	l := log.With().
		Int("projectID", projectID).
		Str("name", name).
		Logger()
	l.Debug().Msg("Reading project access token")

	var pat ProjectAccessToken
	tokens, err := c.ListProjectAccessTokens(projectID)
	if err != nil {
		l.Err(err).
			Msg("Error reading project access token")
		return pat, err
	}

	for _, t := range tokens {
		l.Debug().Msg("Found project access token with matching name")
		if t.Name == name {
			return t, nil
		}
	}

	l.Warn().Msg("Could not find project access token with matching name")
	return pat, ErrNotFound
}

func (c *RollbarApiClient) DeleteProjectAccessToken(token string) error {
	return fmt.Errorf("delete PAT not yet implemented by Rollbar API")
}

// ProjectAccessTokenScope represents the scope of a Rollbar project access token.
type ProjectAccessTokenScope string

// Possible values forproject access token scope
const (
	PATScopeWrite            = ProjectAccessTokenScope("write")
	PATScopeRead             = ProjectAccessTokenScope("read")
	PATScopePostServerItem   = ProjectAccessTokenScope("post_server_item")
	PATScopePostClientServer = ProjectAccessTokenScope("post_client_server")
)

// ProjectAccessTokenArgs encapsulates the required and optional arguments for creating and
// updating Rollbar project access tokens.
type ProjectAccessTokenArgs struct {
	// Required
	ProjectID int `json:"-"`
	Name      string
	Scopes    []ProjectAccessTokenScope
	// Optional - ignored if pointer is nil
	Status               *Status
	RateLimitWindowSize  *int `json:"rate_limit_window_size"`
	RateLimitWindowCount *int `json:"rate_limit_window_count"`
}

// CreateProjectAccessToken creates a Rollbar project access token.
func (c *RollbarApiClient) CreateProjectAccessToken(args ProjectAccessTokenArgs) (ProjectAccessToken, error) {
	l := log.With().
		Interface("args", args).
		Logger()
	var pat ProjectAccessToken

	// Sanity checks
	if args.ProjectID <= 0 {
		err := fmt.Errorf("project ID cannot be blank")
		l.Err(err).Msg("Failed sanity check")
		return pat, err
	}
	if args.Name == "" {
		err := fmt.Errorf("name cannot be blank")
		l.Err(err).Msg("Failed sanity check")
		return pat, err
	}
	if len(args.Scopes) < 1 {
		err := fmt.Errorf("at least one scope must be specified")
		l.Err(err).Msg("Failed sanity check")
		return pat, err
	}

	/*
		// Build request body from arguments
		body := map[string]interface{}{
			"name":  args.Name,
			"scope": args.Scope,
		}
		if args.Status != nil {
			body["status"] = *args.Status
		}
		if args.RateLimitWindowCount != nil {
			body["rate_limit_window_count"] = *args.RateLimitWindowCount
		}
		if args.RateLimitWindowSize != nil {
			body["rate_limit_window_size"] = *args.RateLimitWindowSize
		}
	*/

	u := apiUrl + pathPatCreate
	resp, err := c.resty.R().
		SetPathParams(map[string]string{
			"projectId": strconv.Itoa(args.ProjectID),
		}).
		SetBody(args).
		SetResult(patCreateResponse{}).
		SetError(ErrorResult{}).
		Post(u)
	if err != nil {
		l.Err(err).Msg("Error creating project access token")
		return pat, err
	}
	switch resp.StatusCode() {
	case http.StatusOK, http.StatusCreated:
		// FIXME: currently API returns `200 OK` on successful create; but it
		//  should instead return `201 Created`.
		//  https://github.com/rollbar/terraform-provider-rollbar/issues/8
		r := resp.Result().(*patCreateResponse)
		pat = r.Result
		l.Debug().
			Interface("token", pat).
			Msg("Successfully created new project access token")
		return pat, nil
	case http.StatusUnauthorized:
		l.Warn().Msg("Unauthorized")
		return pat, ErrUnauthorized
	default:
		er := resp.Error().(*ErrorResult)
		l.Error().
			Int("StatusCode", resp.StatusCode()).
			Str("Status", resp.Status()).
			Interface("ErrorResult", er).
			Msg("Error creating project access token")
		return pat, er
	}
}

/*
 * Containers for unmarshalling Rollbar API responses
 */

type patListResponse struct {
	Error  int
	Result []ProjectAccessToken
}

type patCreateResponse struct {
	Error  int
	Result ProjectAccessToken
}
