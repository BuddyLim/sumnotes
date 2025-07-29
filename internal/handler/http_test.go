// internal/handler/http_test.go

package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"

	"main/internal/auth"
	"main/internal/config"
	"main/internal/database"
	"main/internal/model"
)

// MockDB is a mock implementation of the UserStorer interface.
type MockDB struct {
	mock.Mock
}

// Ensure MockDB satisfies the UserStorer interface.
var _ database.UserStore = (*MockDB)(nil)

// MockStore is a mock implementation of the sessions.Store interface.
type MockStore struct {
	mock.Mock
}

type MockProvider struct {
	mock.Mock
}

type MockGothSession struct {
	mock.Mock
}

func (m *MockGothSession) GetAuthURL() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockGothSession) Marshal() string {
	return ""
}

func (m *MockGothSession) Authorize(provider goth.Provider, params goth.Params) (string, error) {
	return "", nil
}

func (m *MockProvider) Name() string { return "mock" }

func (m *MockProvider) SetName(name string) {}

func (m *MockProvider) Debug(debug bool) {}

func (m *MockProvider) BeginAuth(state string) (goth.Session, error) {
	args := m.Called(state)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(goth.Session), args.Error(1)
}
func (m *MockProvider) UnmarshalSession(session string) (goth.Session, error) { return nil, nil }
func (m *MockProvider) FetchUser(session goth.Session) (goth.User, error)     { return goth.User{}, nil }
func (m *MockProvider) RefreshTokenAvailable() bool                           { return true }

func (m *MockProvider) RefreshToken(refreshToken string) (*oauth2.Token, error) {
	args := m.Called(refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*oauth2.Token), args.Error(1)
}

func (m *MockDB) FindUserByEmail(email string) (*model.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockDB) FindUserByID(id string) (*model.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockDB) CreateUser(user *model.User) (*model.User, error) {
	args := m.Called(user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockDB) UpdateUserTokens(userID, accessToken, refreshToken string, tokenExpiry time.Time) error {
	args := m.Called(userID, accessToken, refreshToken, tokenExpiry)
	return args.Error(0)
}

func (m *MockStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	args := m.Called(r, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sessions.Session), args.Error(1)
}

func (m *MockStore) New(r *http.Request, name string) (*sessions.Session, error) {
	args := m.Called(r, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sessions.Session), args.Error(1)
}

func (m *MockStore) Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	args := m.Called(r, w, s)
	return args.Error(0)
}

type MockAuth struct {
	mock.Mock
}

func (m *MockAuth) CompleteUserAuth(w http.ResponseWriter, r *http.Request) (goth.User, error) {
	args := m.Called(r, w)
	// Add a nil check for robustness.
	if args.Get(0) == nil {
		// Return an empty goth.User struct if the mock returns nil.
		return goth.User{}, args.Error(1)
	}
	// Correct the type assertion to match the return type.
	return args.Get(0).(goth.User), args.Error(1)
}

func setupBaseTest() (*httptest.ResponseRecorder, *gin.Engine, *MockDB, *MockStore, *MockProvider, *MockAuth) {
	gin.SetMode(gin.TestMode)

	mockDB := new(MockDB)
	mockStore := new(MockStore)
	mockProvider := new(MockProvider)
	mockAuthenticator := new(MockAuth)

	w := httptest.NewRecorder()
	router := gin.Default()

	return w, router, mockDB, mockStore, mockProvider, mockAuthenticator
}

func TestNew(t *testing.T) {
	t.Run("New Handler", func(t *testing.T) {
		_, _, mockDB, mockStore, mockProvider, mockAuthenticator := setupBaseTest()

		cfg := &config.Config{
			FrontendURL: "example.com",
		}
		h := New(mockDB, mockStore, cfg, mockProvider, mockAuthenticator)

		assert.NotNil(t, h)
		assert.Equal(t, mockDB, h.db)
		assert.Equal(t, mockProvider, h.p)
		assert.Equal(t, mockStore, h.store)
		assert.Equal(t, cfg, h.cfg)
	})
}

func TestSignInWithProvider(t *testing.T) {
	t.Run("Sign in with a provider", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		w, router, mockDB, mockStore, mockProvider, mockAuthenticator := setupBaseTest()

		// Tell gothic to use our mock store
		gothic.Store = mockStore

		// Set up the handler with the mock provider
		h := New(mockDB, mockStore, &config.Config{}, mockProvider, mockAuthenticator)
		router.GET("/auth/:provider", h.SignInWithProvider)

		// Mock the BeginAuth call to return a mock session
		mockSession := new(MockGothSession)
		mockProvider.On("BeginAuth", mock.Anything).Return(mockSession, nil)

		// Mock the GetAuthURL call to return a fake auth URL
		expectedAuthURL := "http://example.com/auth"
		mockSession.On("GetAuthURL").Return(expectedAuthURL, nil)

		// Add the mock provider to goth
		goth.UseProviders(mockProvider)

		// Mock the session store calls from gothic
		session := sessions.NewSession(mockStore, "_gothic_session")
		mockStore.On("New", mock.Anything, "_gothic_session").Return(session, nil)
		mockStore.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// Perform the request
		req, _ := http.NewRequest(http.MethodGet, "/auth/mock", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
		assert.Equal(t, expectedAuthURL, w.Header().Get("Location"))

		mockStore.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})
}

func setupCallBackTest() (*httptest.ResponseRecorder, *gin.Engine, *MockDB, *MockStore, *MockProvider, *MockAuth) {
	w, router, mockDB, mockStore, mockProvider, mockAuthenticator := setupBaseTest()

	h := &Handler{
		db:    mockDB,
		store: mockStore,
		p:     mockProvider,
		auth:  mockAuthenticator,
		cfg: &config.Config{
			FrontendURL: "http://example.com",
		},
	}

	router.GET("/auth/google/callback", h.CallbackHandler)

	return w, router, mockDB, mockStore, mockProvider, mockAuthenticator
}

func TestCallBackHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name           string
		setupMocks     func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider, mockAuthenticator *MockAuth)
		expectedStatus int
		expectedBody   *model.User
	}{
		{
			name: "Callback Failed User Auth",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider, mockAuthenticator *MockAuth) {
				session := sessions.NewSession(mockStore, "sumnotes_session")

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)

				mockAuthenticator.On("CompleteUserAuth", mock.Anything, mock.Anything).Return(nil, errors.New("Error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name: "Callback Failed DB User Search",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider, mockAuthenticator *MockAuth) {
				session := sessions.NewSession(mockStore, "sumnotes_session")

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)

				mockAuthenticator.On("CompleteUserAuth", mock.Anything, mock.Anything).Return(goth.User{
					Email: "abc@abc.com",
				}, nil)

				mockDB.On("FindUserByEmail", "abc@abc.com").Return(nil, errors.New("Error"))

			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name: "Callback Failed DB User nil and create user error",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider, mockAuthenticator *MockAuth) {
				session := sessions.NewSession(mockStore, "sumnotes_session")

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)

				mockAuthenticator.On("CompleteUserAuth", mock.Anything, mock.Anything).Return(goth.User{
					Email: "abc@abc.com",
				}, nil)

				mockDB.On("FindUserByEmail", "abc@abc.com").Return(nil, nil)
				mockDB.On("CreateUser", mock.Anything).Return(nil, errors.New("Error"))

				// mockDB.On("UpdateUserTokens", "abc@abc.com").Return(nil, errors.New("Error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name: "Callback Failed Update Token",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider, mockAuthenticator *MockAuth) {
				session := sessions.NewSession(mockStore, "sumnotes_session")

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)

				mockAuthenticator.On("CompleteUserAuth", mock.Anything, mock.Anything).Return(goth.User{
					Email:        "abc@abc.com",
					AccessToken:  "abc",
					RefreshToken: "def",
					ExpiresAt:    fixedTime,
				}, nil)

				mockDB.On("FindUserByEmail", "abc@abc.com").Return(nil, nil)
				mockDB.On("CreateUser", mock.Anything).Return(&model.User{
					ID: "1",
				}, nil)
				mockDB.On("UpdateUserTokens", "1", "abc", "def", fixedTime).Return(errors.New("Error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name: "Callback Failed Get Session",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider, mockAuthenticator *MockAuth) {

				mockAuthenticator.On("CompleteUserAuth", mock.Anything, mock.Anything).Return(goth.User{
					Email:        "abc@abc.com",
					AccessToken:  "abc",
					RefreshToken: "def",
					ExpiresAt:    fixedTime,
				}, nil)

				mockDB.On("FindUserByEmail", "abc@abc.com").Return(nil, nil)
				mockDB.On("CreateUser", mock.Anything).Return(&model.User{
					ID: "1",
				}, nil)
				mockDB.On("UpdateUserTokens", "1", "abc", "def", fixedTime).Return(nil)

				// session := sessions.NewSession(mockStore, "sumnotes_session")

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(nil, errors.New("Error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name: "Callback Failed Error Session and Error Save",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider, mockAuthenticator *MockAuth) {

				mockAuthenticator.On("CompleteUserAuth", mock.Anything, mock.Anything).Return(goth.User{
					Email:        "abc@abc.com",
					AccessToken:  "abc",
					RefreshToken: "def",
					ExpiresAt:    fixedTime,
				}, nil)

				mockDB.On("FindUserByEmail", "abc@abc.com").Return(nil, nil)
				mockDB.On("CreateUser", mock.Anything).Return(&model.User{
					ID: "1",
				}, nil)
				mockDB.On("UpdateUserTokens", "1", "abc", "def", fixedTime).Return(nil)
				mockDB.On("UpdateUserTokens", "1", "abc", "def", fixedTime).Return(nil)

				session := sessions.NewSession(mockStore, "sumnotes_session")

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
				mockStore.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("session save error"))

			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name: "Callback Success",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider, mockAuthenticator *MockAuth) {

				mockAuthenticator.On("CompleteUserAuth", mock.Anything, mock.Anything).Return(goth.User{
					Email:        "abc@abc.com",
					AccessToken:  "abc",
					RefreshToken: "def",
					ExpiresAt:    fixedTime,
				}, nil)

				mockDB.On("FindUserByEmail", "abc@abc.com").Return(nil, nil)
				mockDB.On("CreateUser", mock.Anything).Return(&model.User{
					ID: "1",
				}, nil)
				mockDB.On("UpdateUserTokens", "1", "abc", "def", fixedTime).Return(nil)
				mockDB.On("UpdateUserTokens", "1", "abc", "def", fixedTime).Return(nil)

				session := sessions.NewSession(mockStore, "sumnotes_session")

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
				mockStore.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			},
			expectedStatus: http.StatusTemporaryRedirect,
			expectedBody:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w, router, mockDB, mockStore, mockProvider, mockAuthenticator := setupCallBackTest()

			tc.setupMocks(mockDB, mockStore, mockProvider, mockAuthenticator)

			req, _ := http.NewRequest(http.MethodGet, "/auth/google/callback?provider=google", nil)
			router.ServeHTTP(w, req)

			if tc.expectedStatus == http.StatusTemporaryRedirect {
				assert.Equal(t, "http://example.com", w.Result().Header.Get("Location"))

			}

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func setupMeTest() (*httptest.ResponseRecorder, *gin.Engine, *MockDB, *MockStore, *MockProvider) {
	w, router, mockDB, mockStore, mockProvider, _ := setupBaseTest()

	h := &Handler{
		db:    mockDB,
		store: mockStore,
		p:     mockProvider,
	}

	router.GET("/me", h.Me)

	return w, router, mockDB, mockStore, mockProvider
}

func TestHandler_Me(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedUser := &model.User{
		ID:          "user-123",
		Name:        "Test User",
		Email:       "test@example.com",
		AvatarURL:   "http://example.com/avatar.png",
		CreatedAt:   time.Now(),
		AccessToken: "abc",
	}

	testCases := []struct {
		name           string
		setupMocks     func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider)
		expectedStatus int
		expectedBody   *model.User
	}{
		{
			name: "Get Me Success",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = "user-123"

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
				mockDB.On("FindUserByID", "user-123").Return(expectedUser, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   expectedUser,
		},
		{
			name: "Get Me Session Error",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = "user-123"

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(nil, errors.New("Failed to Get User Session"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name: "Get Me Session Empty User",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = ""

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   nil,
		},
		{
			name: "Get Me DB Find Error",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = "user-123"

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
				mockDB.On("FindUserByID", "user-123").Return(nil, errors.New("DB Find User Error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name: "Get Me DB No User",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = "user-123"

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
				mockDB.On("FindUserByID", "user-123").Return(nil, nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w, router, mockDB, mockStore, mockProvider := setupMeTest()

			tc.setupMocks(mockDB, mockStore, mockProvider)

			// Perform the request
			req, _ := http.NewRequest(http.MethodGet, "/me", nil)
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedBody != nil {
				var responseBody model.User
				err := json.Unmarshal(w.Body.Bytes(), &responseBody)
				assert.NoError(t, err)

				// We only compare the fields that are expected to be returned.
				// The tokens and expiry are updated in the background.
				assert.Equal(t, tc.expectedBody.ID, responseBody.ID)
				assert.Equal(t, tc.expectedBody.Name, responseBody.Name)
				assert.Equal(t, tc.expectedBody.Email, responseBody.Email)
				assert.Equal(t, tc.expectedBody.AvatarURL, responseBody.AvatarURL)
			}

			// Verify that all mock expectations were met
			mockDB.AssertExpectations(t)
			mockStore.AssertExpectations(t)
			mockProvider.AssertExpectations(t)
		})
	}

}

func TestHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get Me Success", func(t *testing.T) {
		expectedRedirect := "http://test.com"

		_, _, mockDB, mockStore, mockProvider, mockAuthenticator := setupBaseTest()

		cfg := &config.Config{
			FrontendURL: expectedRedirect,
		}
		h := New(mockDB, mockStore, cfg, mockProvider, mockAuthenticator)

		// Setup router
		router := gin.Default()
		router.GET("/success", h.Success)

		// Perform request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/success", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusPermanentRedirect, w.Code)
		assert.Equal(t, expectedRedirect, w.Result().Header.Get("Location"))
	})

}

func setupRefreshTest() (*httptest.ResponseRecorder, *gin.Engine, *MockDB, *MockStore, *MockProvider) {
	w, router, mockDB, mockStore, mockProvider, mockAuthenticator := setupBaseTest()

	cfg := &config.Config{}
	h := New(mockDB, mockStore, cfg, mockProvider, mockAuthenticator)

	router.GET("/refresh", h.Refresh)

	return w, router, mockDB, mockStore, mockProvider
}

func TestHandler_Refresh(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	expectedUser := &model.User{
		ID:           "user-123",
		Name:         "Test User",
		Email:        "test@example.com",
		AvatarURL:    "http://example.com/avatar.png",
		RefreshToken: "old-refresh-token",
		CreatedAt:    fixedTime,
	}

	// Define the new token details
	newToken := &oauth2.Token{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		Expiry:       fixedTime.Add(1 * time.Hour),
	}

	testCases := []struct {
		name           string
		setupMocks     func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider)
		expectedStatus int
		expectedBody   *model.User
	}{
		{
			name: "Get Refresh Success",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = "user-123"

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
				mockDB.On("FindUserByID", "user-123").Return(expectedUser, nil)
				mockProvider.On("RefreshToken", expectedUser.RefreshToken).Return(newToken, nil)
				mockDB.On("UpdateUserTokens", "user-123", newToken.AccessToken, newToken.RefreshToken, newToken.Expiry).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   expectedUser,
		},
		{
			name: "Get Refresh Session Failure",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = "user-123"

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(nil, errors.New("Failed to get user session"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name: "Get Refresh No User Session",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = ""

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   nil,
		},
		{
			name: "Get Refresh DB User Error",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = "user-123"

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)

				mockDB.On("FindUserByID", "user-123").Return(nil, errors.New("Failed to get user in DB"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name: "Get Refresh DB No User",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = "user-123"

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)

				mockDB.On("FindUserByID", "user-123").Return(nil, nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   nil,
		},
		{
			name: "Refresh fails and session is cleared",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = "user-123"

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
				mockDB.On("FindUserByID", "user-123").Return(expectedUser, nil)
				mockProvider.On("RefreshToken", expectedUser.RefreshToken).Return(nil, auth.ErrRefreshFailed)

				// Expect Save to be called to clear the session
				mockStore.On("Save", mock.Anything, mock.Anything, mock.MatchedBy(func(s *sessions.Session) bool {
					return s.Options.MaxAge == -1
				})).Return(nil)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   nil,
		},
		{
			name: "Refresh fails and session save fails",
			setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
				session := sessions.NewSession(mockStore, "sumnotes_session")
				session.Values["user_id"] = "user-123"

				mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
				mockDB.On("FindUserByID", "user-123").Return(expectedUser, nil)
				mockProvider.On("RefreshToken", expectedUser.RefreshToken).Return(nil, auth.ErrRefreshFailed)

				// Expect Save to be called and fail
				mockStore.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("session save error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w, router, mockDB, mockStore, mockProvider := setupRefreshTest()

			tc.setupMocks(mockDB, mockStore, mockProvider)

			// Perform the request
			req, _ := http.NewRequest(http.MethodGet, "/refresh", nil)
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedBody != nil {
				var responseBody model.User
				err := json.Unmarshal(w.Body.Bytes(), &responseBody)
				assert.NoError(t, err)

				// We only compare the fields that are expected to be returned.
				// The tokens and expiry are updated in the background.
				assert.Equal(t, tc.expectedBody.ID, responseBody.ID)
				assert.Equal(t, tc.expectedBody.Name, responseBody.Name)
				assert.Equal(t, tc.expectedBody.Email, responseBody.Email)
				assert.Equal(t, tc.expectedBody.AvatarURL, responseBody.AvatarURL)
			}

			// Verify that all mock expectations were met
			mockDB.AssertExpectations(t)
			mockStore.AssertExpectations(t)
			mockProvider.AssertExpectations(t)
		})
	}
}
