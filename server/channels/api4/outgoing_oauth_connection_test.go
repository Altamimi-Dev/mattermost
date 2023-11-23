// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
package api4

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/mattermost/mattermost/server/v8/channels/web"
	"github.com/mattermost/mattermost/server/v8/einterfaces/mocks"
	"github.com/stretchr/testify/require"
)

func outgoingOauthConnectionsCleanup(t *testing.T, th *TestHelper) {
	// Remove all connections
	conns, errCleanup := th.App.Srv().Store().OutgoingOAuthConnection().GetConnections(th.Context, model.OutgoingOAuthConnectionGetConnectionsFilter{})
	require.NoError(t, errCleanup)

	for _, c := range conns {
		require.NoError(t, th.App.Srv().Store().OutgoingOAuthConnection().DeleteConnection(th.Context, c.Id))
	}
}

// Client tests

func TestOutgoingOAuthConnectionGet(t *testing.T) {
	t.Run("No license returns 501", func(t *testing.T) {
		os.Setenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTION", "true")
		defer os.Unsetenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTION")
		th := Setup(t).InitBasic()
		defer th.TearDown()

		outgoingOauthIface := &mocks.OutgoingOAuthConnectionInterface{}
		outgoingOauthImpl := th.App.Srv().OutgoingOAuthConnection
		defer func() {
			th.App.Srv().OutgoingOAuthConnection = outgoingOauthImpl
		}()
		th.App.Srv().OutgoingOAuthConnection = outgoingOauthIface

		th.Client.Login(context.Background(), th.BasicUser.Email, th.BasicUser.Password)

		connections, response, err := th.Client.GetOutgoingOAuthConnections(context.Background(), "", 10)
		require.Error(t, err)
		require.Nil(t, connections)
		require.Equal(t, 501, response.StatusCode)
	})

	t.Run("license but no feature flag return 501", func(t *testing.T) {
		th := Setup(t).InitBasic()
		defer th.TearDown()

		outgoingOauthIface := &mocks.OutgoingOAuthConnectionInterface{}
		outgoingOauthImpl := th.App.Srv().OutgoingOAuthConnection
		defer func() {
			th.App.Srv().OutgoingOAuthConnection = outgoingOauthImpl
		}()
		th.App.Srv().OutgoingOAuthConnection = outgoingOauthIface
		license := model.NewTestLicenseSKU(model.LicenseShortSkuEnterprise, "outgoing_oauth_connections")
		license.Id = "test-license-id"
		th.App.Srv().SetLicense(license)
		th.App.Srv().RemoveLicense()

		th.Client.Login(context.Background(), th.BasicUser.Email, th.BasicUser.Password)

		connections, response, err := th.Client.GetOutgoingOAuthConnections(context.Background(), "", 10)
		require.Error(t, err)
		require.Nil(t, connections)
		require.Equal(t, 501, response.StatusCode)
	})
}

func TestListOutgoingOAutConnection(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		os.Setenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS", "true")
		defer os.Unsetenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS")
		th := Setup(t).InitBasic()
		defer th.TearDown()
		defer th.App.Srv().RemoveLicense()
		defer outgoingOauthConnectionsCleanup(t, th)

		license := model.NewTestLicenseSKU(model.LicenseShortSkuEnterprise, "outgoing_oauth_connections")
		license.Id = "test-license-id"
		th.App.Srv().SetLicense(license)

		outgoingOauthIface := &mocks.OutgoingOAuthConnectionInterface{}
		outgoingOauthIface.Mock.On("GetConnections", mock.Anything, mock.Anything).Return([]*model.OutgoingOAuthConnection{}, nil)
		outgoingOauthIface.Mock.On("SanitizeConnections", mock.Anything)

		outgoingOauthImpl := th.App.Srv().OutgoingOAuthConnection
		defer func() {
			th.App.Srv().OutgoingOAuthConnection = outgoingOauthImpl
		}()
		th.App.Srv().OutgoingOAuthConnection = outgoingOauthIface

		th.Client.Login(context.Background(), th.BasicUser.Email, th.BasicUser.Password)

		connections, response, err := th.Client.GetOutgoingOAuthConnections(context.Background(), "", 10)
		require.NoError(t, err)

		require.Equal(t, 200, response.StatusCode)
		require.Equal(t, 0, len(connections))
	})

	t.Run("return result", func(t *testing.T) {
		os.Setenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS", "true")
		defer os.Unsetenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS")
		th := Setup(t).InitBasic()
		defer th.TearDown()
		defer th.App.Srv().RemoveLicense()
		defer outgoingOauthConnectionsCleanup(t, th)

		conn := &model.OutgoingOAuthConnection{
			Name: "test",
		}

		th.App.Srv().Store().OutgoingOAuthConnection().SaveConnection(th.Context, conn)

		license := model.NewTestLicenseSKU(model.LicenseShortSkuEnterprise, "outgoing_oauth_connections")
		license.Id = "test-license-id"
		th.App.Srv().SetLicense(license)

		outgoingOauthIface := &mocks.OutgoingOAuthConnectionInterface{}
		outgoingOauthIface.Mock.On("GetConnections", mock.Anything, mock.Anything).Return([]*model.OutgoingOAuthConnection{conn}, nil)
		outgoingOauthIface.Mock.On("SanitizeConnections", mock.Anything)

		outgoingOauthImpl := th.App.Srv().OutgoingOAuthConnection
		defer func() {
			th.App.Srv().OutgoingOAuthConnection = outgoingOauthImpl
		}()
		th.App.Srv().OutgoingOAuthConnection = outgoingOauthIface

		th.Client.Login(context.Background(), th.BasicUser.Email, th.BasicUser.Password)

		connections, response, err := th.Client.GetOutgoingOAuthConnections(context.Background(), "", 10)
		require.NoError(t, err)

		require.Equal(t, 200, response.StatusCode)
		require.Equal(t, 1, len(connections))
	})
}

func TestGetOutgoingOauthConnection(t *testing.T) {
	t.Run("return result", func(t *testing.T) {
		os.Setenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS", "true")
		defer os.Unsetenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS")
		th := Setup(t).InitBasic()
		defer th.TearDown()
		defer th.App.Srv().RemoveLicense()
		defer outgoingOauthConnectionsCleanup(t, th)

		conn := &model.OutgoingOAuthConnection{
			Name: "test",
		}

		th.App.Srv().Store().OutgoingOAuthConnection().SaveConnection(th.Context, conn)

		license := model.NewTestLicenseSKU(model.LicenseShortSkuEnterprise, "outgoing_oauth_connections")
		license.Id = "test-license-id"
		th.App.Srv().SetLicense(license)

		outgoingOauthIface := &mocks.OutgoingOAuthConnectionInterface{}
		outgoingOauthIface.Mock.On("GetConnection", mock.Anything, mock.Anything).Return(conn, nil)
		outgoingOauthIface.Mock.On("SanitizeConnection", mock.Anything)

		outgoingOauthImpl := th.App.Srv().OutgoingOAuthConnection
		defer func() {
			th.App.Srv().OutgoingOAuthConnection = outgoingOauthImpl
		}()
		th.App.Srv().OutgoingOAuthConnection = outgoingOauthIface

		th.Client.Login(context.Background(), th.BasicUser.Email, th.BasicUser.Password)

		connection, response, err := th.Client.GetOutgoingOAuthConnection(context.Background(), conn.Id)
		require.NoError(t, err)

		require.Equal(t, 200, response.StatusCode)
		require.NotNil(t, connection)
		require.Equal(t, conn.Id, connection.Id)
	})
}

// API tests

func TestEnsureOutgoingOAuthConnectionInterface(t *testing.T) {
	t.Run("no feature flag, no interface, no license", func(t *testing.T) {
		th := Setup(t).InitBasic()
		defer th.TearDown()
		c := &Context{}
		c.AppContext = th.Context
		c.App = th.App
		c.Logger = th.App.Srv().Log()

		_, valid := ensureOutgoingOAuthConnectionInterface(c, "api")
		require.False(t, valid)
	})

	t.Run("feature flag, no interface, no license", func(t *testing.T) {
		os.Setenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS", "true")
		defer os.Unsetenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS")
		th := Setup(t).InitBasic()
		defer th.TearDown()
		c := &Context{}
		c.AppContext = th.Context
		c.App = th.App
		c.Logger = th.App.Srv().Log()

		_, valid := ensureOutgoingOAuthConnectionInterface(c, "api")
		require.False(t, valid)
	})

	t.Run("feature flag, interface defined, no license", func(t *testing.T) {
		os.Setenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS", "true")
		defer os.Unsetenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS")
		th := Setup(t).InitBasic()
		defer th.TearDown()

		outgoingOauthIface := &mocks.OutgoingOAuthConnectionInterface{}
		outgoingOauthImpl := th.App.Srv().OutgoingOAuthConnection
		defer func() {
			th.App.Srv().OutgoingOAuthConnection = outgoingOauthImpl
		}()
		th.App.Srv().OutgoingOAuthConnection = outgoingOauthIface

		c := &Context{}
		c.AppContext = th.Context
		c.App = th.App
		c.Logger = th.App.Srv().Log()

		_, valid := ensureOutgoingOAuthConnectionInterface(c, "api")
		require.False(t, valid)
	})

	t.Run("feature flag, interface defined, valid license", func(t *testing.T) {
		os.Setenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS", "true")
		defer os.Unsetenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS")
		th := Setup(t).InitBasic()
		defer th.TearDown()

		license := model.NewTestLicenseSKU(model.LicenseShortSkuEnterprise, "outgoing_oauth_connections")
		license.Id = "test-license-id"
		th.App.Srv().SetLicense(license)
		defer th.App.Srv().RemoveLicense()

		outgoingOauthIface := &mocks.OutgoingOAuthConnectionInterface{}
		outgoingOauthImpl := th.App.Srv().OutgoingOAuthConnection
		defer func() {
			th.App.Srv().OutgoingOAuthConnection = outgoingOauthImpl
		}()
		th.App.Srv().OutgoingOAuthConnection = outgoingOauthIface

		c := &Context{}
		c.AppContext = th.Context
		c.App = th.App
		c.Logger = th.App.Srv().Log()

		svc, valid := ensureOutgoingOAuthConnectionInterface(c, "api")
		require.True(t, valid)
		require.NotNil(t, svc)
	})
}

func TestOutgoingOAuthConnectionAPIHandlers(t *testing.T) {
	os.Setenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS", "true")
	defer os.Unsetenv("MM_FEATUREFLAGS_OUTGOINGOAUTHCONNECTIONS")
	th := Setup(t).InitBasic()
	defer th.TearDown()

	license := model.NewTestLicenseSKU(model.LicenseShortSkuEnterprise, "outgoing_oauth_connections")
	license.Id = "test-license-id"
	th.App.Srv().SetLicense(license)
	defer th.App.Srv().RemoveLicense()

	outgoingOauthIface := &mocks.OutgoingOAuthConnectionInterface{}
	outgoingOauthImpl := th.App.Srv().OutgoingOAuthConnection
	defer func() {
		th.App.Srv().OutgoingOAuthConnection = outgoingOauthImpl
	}()
	th.App.Srv().OutgoingOAuthConnection = outgoingOauthIface

	c := &Context{}
	c.AppContext = th.Context
	c.App = th.App
	c.Logger = th.App.Srv().Log()

	conn := &model.OutgoingOAuthConnection{
		Id:   "test",
		Name: "test",
	}

	t.Run("getOutgoingOAuthConnection", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		c.Params = &web.Params{
			OutgoingOAuthConnectionID: conn.Id,
		}

		outgoingOauthIface.Mock.On("GetConnection", th.Context, c.Params.OutgoingOAuthConnectionID).Return(conn, nil)
		outgoingOauthIface.Mock.On("SanitizeConnection", mock.Anything)

		httpRecorder := httptest.NewRecorder()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			getOutgoingOAuthConnection(c, w, r)
		})

		handler.ServeHTTP(httpRecorder, req)

		require.Equal(t, http.StatusOK, httpRecorder.Code)
		require.NotEmpty(t, httpRecorder.Body.String())

		var buf bytes.Buffer
		require.NoError(t, json.NewEncoder(&buf).Encode(conn))
	})

	t.Run("listOutgoingOAuthConnections", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		conns := []*model.OutgoingOAuthConnection{conn}

		outgoingOauthIface.Mock.On("GetConnections", th.Context, mock.Anything).Return(conns, nil)
		outgoingOauthIface.Mock.On("SanitizeConnections", mock.Anything)

		httpRecorder := httptest.NewRecorder()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			listOutgoingOAuthConnections(c, w, r)
		})

		handler.ServeHTTP(httpRecorder, req)

		require.Equal(t, http.StatusOK, httpRecorder.Code)
		require.NotEmpty(t, httpRecorder.Body.String())

		var buf bytes.Buffer
		require.NoError(t, json.NewEncoder(&buf).Encode(conn))
	})
}
