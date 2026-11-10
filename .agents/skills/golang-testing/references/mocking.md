# Mocking and Test Fixtures

## Table of Contents

- [Mocks with testify/mock](#mocks-with-testifymock)
- [Organizing mocks](#organizing-mocks)
- [Test fixtures](#test-fixtures)
- [Time mocking](#time-mocking)

## Mocks with testify/mock

Define small interfaces at the consumer and mock those — never concrete types or libraries you do not own.

> The full testify/mock API (matchers, call modifiers, verification) lives in the `fabianoflorentino/golang-agent-skills@golang-stretchr-testify` skill.

```go
// Interface the consumer depends on
type Database interface {
    GetUser(id string) (*User, error)
    CreateUser(user *User) error
}

// Mock implementation
type MockDatabase struct{ mock.Mock }

func (m *MockDatabase) GetUser(id string) (*User, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

func (m *MockDatabase) CreateUser(user *User) error {
    return m.Called(user).Error(0)
}

func TestService_GetUser(t *testing.T) {
    is := assert.New(t)

    mockDB := new(MockDatabase)
    service := NewService(mockDB)

    expectedUser := &User{ID: "1", Name: "John"}
    mockDB.On("GetUser", "1").Return(expectedUser, nil)

    user, err := service.GetUser("1")

    is.NoError(err)
    is.Equal(expectedUser, user)
    mockDB.AssertExpectations(t)
}

func TestService_GetUser_NotFound(t *testing.T) {
    is := assert.New(t)

    mockDB := new(MockDatabase)
    service := NewService(mockDB)

    mockDB.On("GetUser", "999").Return(nil, ErrNotFound)

    user, err := service.GetUser("999")

    is.Error(err)
    is.ErrorIs(err, ErrNotFound)
    is.Nil(user)
    mockDB.AssertExpectations(t)
}
```

Use `ErrorIs` for wrapped sentinel errors instead of `Equal`.

## Organizing mocks

Keep interfaces next to the code that consumes them, and mocks in the package's `_test.go` file so they compile only during tests:

```go
// user_service.go
type UserService struct {
    db    Database
    email EmailService
}

type Database interface {
    GetUser(id string) (*User, error)
    CreateUser(user *User) error
}

type EmailService interface {
    SendWelcomeEmail(to string) error
}
```

```go
// user_service_test.go
package mypackage_test

type MockDatabase struct{ mock.Mock }

func (m *MockDatabase) GetUser(id string) (*mypackage.User, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*mypackage.User), args.Error(1)
}

func (m *MockDatabase) CreateUser(user *mypackage.User) error {
    return m.Called(user).Error(0)
}

type MockEmailService struct{ mock.Mock }

func (m *MockEmailService) SendWelcomeEmail(to string) error {
    return m.Called(to).Error(0)
}

func TestUserService_CreateUser(t *testing.T) {
    mockDB := new(MockDatabase)
    mockEmail := new(MockEmailService)
    service := mypackage.NewUserService(mockDB, mockEmail)

    user := &mypackage.User{Name: "Test", Email: "test@example.com"}
    mockDB.On("CreateUser", user).Return(nil)
    mockEmail.On("SendWelcomeEmail", "test@example.com").Return(nil)

    err := service.CreateUser(user)

    assert.NoError(t, err)
    mockDB.AssertExpectations(t)
    mockEmail.AssertExpectations(t)
}
```

Generated mocks exist too (`mockery`, `go-mockgen`) — worth it once the mock body outgrows a delegation line per method.

## Test fixtures

Share reusable test data from a fixtures package:

```go
package fixtures

var (
    DefaultUser = &User{
        ID:        "user-123",
        Name:      "Jane Doe",
        Email:     "jane@example.com",
        CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    }

    AdminUser = &User{
        ID:        "admin-1",
        Name:      "Admin User",
        Email:     "admin@example.com",
        Role:      "admin",
        CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    }
)

func NewUser(name, email string) *User {
    return &User{
        ID:        "user-" + uuid.New().String(),
        Name:      name,
        Email:     email,
        CreatedAt: time.Now(),
    }
}
```

Keep fixtures immutable — a test that mutates shared fixture state leaks into whichever test runs next.

## Time mocking

`clockwork` replaces wall-clock time in the code under test, so time-dependent behavior is deterministic and needs no sleeps:

```go
import "github.com/jonboulle/clockwork"

func TestScheduler_AddJob(t *testing.T) {
    is := assert.New(t)

    fakeClock := clockwork.NewFakeClock()
    scheduler := NewScheduler(fakeClock)

    job := &Job{ID: "1", RunAt: time.Now().Add(1 * time.Hour)}
    scheduler.AddJob(job)

    is.Equal(1, scheduler.PendingCount())

    fakeClock.Advance(2 * time.Hour) // jump forward

    is.Equal(0, scheduler.PendingCount())
}
```

Install with `go get github.com/jonboulle/clockwork`. The constructor takes a clock interface, so production passes `clockwork.NewRealClock()` and tests pass a fake. For Go 1.25+, `testing/synctest` is an alternative for goroutine-heavy time logic.
