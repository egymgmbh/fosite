// Copyright © 2017 Aeneas Rekkas <aeneas+oss@aeneas.io>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openid

import (
	"testing"
	"time"

	"fmt"

	"github.com/golang/mock/gomock"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/storage"
	"github.com/ory/fosite/token/jwt"
	"github.com/stretchr/testify/assert"
)

func TestImplicit_HandleAuthorizeEndpointRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	aresp := fosite.NewAuthorizeResponse()
	areq := fosite.NewAuthorizeRequest()
	areq.Session = new(fosite.DefaultSession)

	h := OpenIDConnectImplicitHandler{
		AuthorizeImplicitGrantTypeHandler: &oauth2.AuthorizeImplicitGrantTypeHandler{
			AccessTokenLifespan: time.Hour,
			AccessTokenStrategy: hmacStrategy,
			AccessTokenStorage:  storage.NewMemoryStore(),
		},
		IDTokenHandleHelper: &IDTokenHandleHelper{
			IDTokenStrategy: idStrategy,
		},
		ScopeStrategy: fosite.HierarchicScopeStrategy,
	}
	for k, c := range []struct {
		description string
		setup       func()
		expectErr   error
		check       func()
	}{
		{
			description: "should not do anything because request requirements are not met",
			setup:       func() {},
		},
		{
			description: "should not do anything because request requirements are not met",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"id_token"}
				areq.State = "foostate"
			},
		},
		{
			description: "should not do anything because request requirements are not met",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"token", "id_token"}
			},
		},
		{
			description: "should not do anything because request requirements are not met",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{}
				areq.GrantedScopes = fosite.Arguments{"openid"}
			},
		},
		{
			description: "should not do anything because request requirements are not met",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"token", "id_token"}
				areq.Scopes = fosite.Arguments{"openid"}
				areq.Client = &fosite.DefaultClient{
					GrantTypes:    fosite.Arguments{},
					ResponseTypes: fosite.Arguments{},
					Scopes:        []string{"openid", "fosite"},
				}
			},
			expectErr: fosite.ErrInvalidGrant,
		},
		{
			description: "should not do anything because request requirements are not met",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"token", "id_token"}
				areq.Scopes = fosite.Arguments{"openid"}
				areq.Client = &fosite.DefaultClient{
					GrantTypes:    fosite.Arguments{"implicit"},
					ResponseTypes: fosite.Arguments{},
					Scopes:        []string{"openid", "fosite"},
				}
			},
			expectErr: fosite.ErrInvalidGrant,
		},
		{
			description: "should fail because session not set",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"id_token"}
				areq.Scopes = fosite.Arguments{"openid"}
				areq.Client = &fosite.DefaultClient{
					GrantTypes:    fosite.Arguments{"implicit"},
					ResponseTypes: fosite.Arguments{"token", "id_token"},
					Scopes:        []string{"openid", "fosite"},
				}
			},
			expectErr: ErrInvalidSession,
		},
		{
			description: "should fail because nonce not set",
			setup: func() {
				areq.Session = &DefaultSession{
					Claims: &jwt.IDTokenClaims{
						Subject: "peter",
					},
					Headers: &jwt.Headers{},
				}
				areq.Form.Add("nonce", "some-random-foo-nonce-wow")
			},
		},
		{
			description: "should pass",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"id_token"}
			},
			check: func() {
				assert.NotEmpty(t, aresp.GetFragment().Get("id_token"))
				assert.NotEmpty(t, aresp.GetFragment().Get("state"))
				assert.Empty(t, aresp.GetFragment().Get("access_token"))
			},
		},
		{
			description: "should pass",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"token", "id_token"}
			},
			check: func() {
				assert.NotEmpty(t, aresp.GetFragment().Get("id_token"))
				assert.NotEmpty(t, aresp.GetFragment().Get("state"))
				assert.NotEmpty(t, aresp.GetFragment().Get("access_token"))
			},
		},
		{
			description: "should pass",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"id_token", "token"}
				areq.Scopes = fosite.Arguments{"fosite", "openid"}
			},
			check: func() {
				assert.NotEmpty(t, aresp.GetFragment().Get("id_token"))
				assert.NotEmpty(t, aresp.GetFragment().Get("state"))
				assert.NotEmpty(t, aresp.GetFragment().Get("access_token"))
			},
		},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			c.setup()
			err := h.HandleAuthorizeEndpointRequest(nil, areq, aresp)

			if c.expectErr != nil {
				assert.EqualError(t, err, c.expectErr.Error())
			} else {
				assert.NoError(t, err)
				if c.check != nil {
					c.check()
				}
			}
		})
	}
}
