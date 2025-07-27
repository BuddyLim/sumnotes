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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"

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

func (m *MockProvider) Name() string                                          { return "mock" }
func (m *MockProvider) SetName(name string)                                   {}
func (m *MockProvider) Debug(debug bool)                                      {}
func (m *MockProvider) BeginAuth(state string) (goth.Session, error)          { return nil, nil }
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

func TestHandler_Me(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get Me Success", func(t *testing.T) {
		// Setup
		mockDB := new(MockDB)
		mockStore := new(MockStore)

		// Expected user
		expectedUser := &model.User{
			ID:        "user-123",
			Name:      "Test User",
			Email:     "test@example.com",
			AvatarURL: "http://example.com/avatar.png",
			CreatedAt: time.Now(),
		}

		// Configure mocks
		session := sessions.NewSession(mockStore, "sumnotes_session")
		session.Values["user_id"] = "user-123"

		mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
		mockDB.On("FindUserByID", "user-123").Return(expectedUser, nil)

		// Create handler with mocks
		h := &Handler{
			db:    mockDB,
			store: mockStore,
		}

		// Setup router
		router := gin.Default()
		router.GET("/me", h.Me)

		// Perform request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/me", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var responseBody map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &responseBody)

		assert.Equal(t, expectedUser.ID, responseBody["ID"])
		assert.Equal(t, expectedUser.Name, responseBody["Name"])
		assert.Equal(t, expectedUser.Email, responseBody["Email"])

		mockDB.AssertExpectations(t)
		mockStore.AssertExpectations(t)
	})

	t.Run("Get Me No Db Entry", func(t *testing.T) {
		// Setup
		mockDB := new(MockDB)
		mockStore := new(MockStore)

		// Configure mocks
		session := sessions.NewSession(mockStore, "sumnotes_session")
		session.Values["user_id"] = "user-124"

		mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
		mockDB.On("FindUserByID", "user-124").Return(nil, nil)

		// Create handler with mocks
		h := &Handler{
			db:    mockDB,
			store: mockStore,
		}

		// Setup router
		router := gin.Default()
		router.GET("/me", h.Me)

		// Perform request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/me", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusNotFound, w.Code)

		mockDB.AssertExpectations(t)
		mockStore.AssertExpectations(t)
	})

	t.Run("Get Me Error", func(t *testing.T) {
		// Setup
		mockDB := new(MockDB)
		mockStore := new(MockStore)

		// Configure mocks
		session := sessions.NewSession(mockStore, "sumnotes_session")
		session.Values["user_id"] = "user-124"

		mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
		mockDB.On("FindUserByID", "user-124").Return(nil, errors.New("error"))

		// Create handler with mocks
		h := &Handler{
			db:    mockDB,
			store: mockStore,
		}

		// Setup router
		router := gin.Default()
		router.GET("/me", h.Me)

		// Perform request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/me", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockDB.AssertExpectations(t)
		mockStore.AssertExpectations(t)
	})

	t.Run("Get Me No Session", func(t *testing.T) {
		// Setup
		mockDB := new(MockDB)
		mockStore := new(MockStore)

		// Configure mocks
		session := sessions.NewSession(mockStore, "sumnotes_session")
		session.Values["user_id"] = ""

		mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
		mockDB.On("FindUserByID", "").Return(nil, nil)

		// Create handler with mocks
		h := &Handler{
			db:    mockDB,
			store: mockStore,
		}

		// Setup router
		router := gin.Default()
		router.GET("/me", h.Me)

		// Perform request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/me", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	})

	t.Run("Get Me Session Error", func(t *testing.T) {
		// Setup
		mockDB := new(MockDB)
		mockStore := new(MockStore)

		// Configure mocks
		session := sessions.NewSession(mockStore, "sumnotes_session")
		session.Values["user_id"] = ""

		mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, errors.New("error"))
		mockDB.On("FindUserByID", "").Return(nil, nil)

		// Create handler with mocks
		h := &Handler{
			db:    mockDB,
			store: mockStore,
		}

		// Setup router
		router := gin.Default()
		router.GET("/me", h.Me)

		// Perform request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/me", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusInternalServerError, w.Code)

	})

	// ... Add other test cases for failure scenarios here
}

func TestHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get Me Success", func(t *testing.T) {
		expectedRedirect := "http://test.com"

		// Create handler with mocks
		h := &Handler{

			cfg: &config.Config{
				FrontendURL: expectedRedirect,
			},
		}

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

func setupBaseTest() (*httptest.ResponseRecorder, *gin.Engine, *MockDB, *MockStore, *MockProvider) {
	gin.SetMode(gin.TestMode)

	mockDB := new(MockDB)
	mockStore := new(MockStore)
	mockProvider := new(MockProvider)

	w := httptest.NewRecorder()
	router := gin.Default()

	return w, router, mockDB, mockStore, mockProvider
}

/**
TODO: Table driven test
setup proper test helpers
*/

func setupRefreshTest(t *testing.T) (*httptest.ResponseRecorder, *gin.Engine, *MockDB, *MockStore, *MockProvider) {
	w, router, mockDB, mockStore, mockProvider := setupBaseTest()

	h := &Handler{
		db:    mockDB,
		store: mockStore,
		p:     mockProvider,
	}

	router.GET("/refresh", h.Refresh)

	return w, router, mockDB, mockStore, mockProvider
}

func TestHandler_Refresh(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// testCases := []struct {
	// 	name           string
	// 	setupMocks     func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider)
	// 	expectedStatus int
	// 	expectedBody   string
	// }{
	// 	{
	// 		name: "Success",
	// 		setupMocks: func(mockDB *MockDB, mockStore *MockStore, mockProvider *MockProvider) {
	// 			// Expected user
	// 			expectedUser := &model.User{
	// 				ID:           "user-123",
	// 				Name:         "Test User",
	// 				Email:        "test@example.com",
	// 				AvatarURL:    "http://example.com/avatar.png",
	// 				CreatedAt:    time.Now(),
	// 				RefreshToken: "abc",
	// 			}

	// 			// Configure mocks
	// 			session := sessions.NewSession(mockStore, "sumnotes_session")
	// 			session.Values["user_id"] = "user-123"

	// 			now := time.Now()

	// 			mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
	// 			mockDB.On("FindUserByID", "user-123").Return(expectedUser, nil)
	// 			mockDB.On("UpdateUserTokens", "user-123", "abc", "def", now).Return(nil)
	// 			mockProvider.On("RefreshToken", expectedUser.RefreshToken).Return(&oauth2.Token{
	// 				AccessToken:  "abc",
	// 				RefreshToken: "def",
	// 				Expiry:       now,
	// 			}, nil)
	// 		},
	// 		expectedStatus: http.StatusOK,
	// 		expectedBody:   `{"ID":"user-123", ...}`,
	// 	},
	// }

	// for _, tc := range testCases {
	// 	t.Run(tc.name, func(t *testing.T) {
	// 		mockDB := new(MockDB)
	// 		mockStore := new(MockStore)
	// 		mockProvider := new(MockProvider)

	// 		tc.setupMocks(mockDB, mockStore)
	// 		// ... execute the test with tc.setupMocks and assert against tc.expectedStatus and tc.expectedBody
	// 	})
	// }

	t.Run("Get Refresh Success", func(t *testing.T) {
		// Setup
		mockDB := new(MockDB)
		mockStore := new(MockStore)
		mockProvider := new(MockProvider)

		// Expected user
		expectedUser := &model.User{
			ID:           "user-123",
			Name:         "Test User",
			Email:        "test@example.com",
			AvatarURL:    "http://example.com/avatar.png",
			CreatedAt:    time.Now(),
			RefreshToken: "abc",
		}

		// Configure mocks
		session := sessions.NewSession(mockStore, "sumnotes_session")
		session.Values["user_id"] = "user-123"

		now := time.Now()

		mockStore.On("Get", mock.Anything, "sumnotes_session").Return(session, nil)
		mockDB.On("FindUserByID", "user-123").Return(expectedUser, nil)
		mockDB.On("UpdateUserTokens", "user-123", "abc", "def", now).Return(nil)
		mockProvider.On("RefreshToken", expectedUser.RefreshToken).Return(&oauth2.Token{
			AccessToken:  "abc",
			RefreshToken: "def",
			Expiry:       now,
		}, nil)

		// Create handler with mocks
		h := &Handler{
			db:    mockDB,
			store: mockStore,
			p:     mockProvider,
		}

		// Setup router
		router := gin.Default()
		router.GET("/refresh", h.Refresh)

		// Perform request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/refresh", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var responseBody map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &responseBody)

		assert.Equal(t, expectedUser.ID, responseBody["ID"])
		assert.Equal(t, expectedUser.Name, responseBody["Name"])
		assert.Equal(t, expectedUser.Email, responseBody["Email"])

		mockDB.AssertExpectations(t)
		mockStore.AssertExpectations(t)
	})

}
