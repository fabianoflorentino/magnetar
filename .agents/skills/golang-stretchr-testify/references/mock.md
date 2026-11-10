# testify/mock — Reference

Embed `mock.Mock` to stand in for an interface, implement each method by delegating to `m.Called(...)`, then prove the calls were made with `AssertExpectations(t)`.

## Quick example

```go
type MockSender struct{ mock.Mock }

func (m *MockSender) Send(ctx context.Context, to string, msg Message) error {
    return m.Called(ctx, to, msg).Error(0)
}

func TestOrderService_Place(t *testing.T) {
    is := assert.New(t)
    m := new(MockSender)
    m.On("Send", mock.Anything, "buyer@example.com", mock.AnythingOfType("Message")).Return(nil)

    err := NewOrderService(m).Place(context.Background(), order)

    is.NoError(err)
    m.AssertExpectations(t)
}
```

## Defining a mock

Define the interface at the consumer, then mirror it:

```go
type NotificationSender interface {
    Send(ctx context.Context, to string, msg Message) error
    BatchSend(ctx context.Context, recipients []string, msg Message) (int, error)
}

type MockNotificationSender struct{ mock.Mock }

func (m *MockNotificationSender) Send(ctx context.Context, to string, msg Message) error {
    return m.Called(ctx, to, msg).Error(0)
}

func (m *MockNotificationSender) BatchSend(ctx context.Context, recipients []string, msg Message) (int, error) {
    args := m.Called(ctx, recipients, msg)
    return args.Int(0), args.Error(1)
}
```

Position indexed accessors (`args.Int(0)`, `args.Error(1)`) read the return values in declaration order — `args.Get(n)` for anything else.

## Argument matchers

```go
// Any value at all
m.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)

// By type name
m.On("Send", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)

// Custom predicate
m.On("Send", mock.Anything, mock.MatchedBy(func(to string) bool {
    return strings.HasSuffix(to, "@example.com")
}), mock.Anything).Return(nil)
```

## Call modifiers

```go
m.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()   // exactly 1 call
m.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(3) // exactly 3 calls
m.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()  // optional

// Side effects via Run
m.On("Send", mock.Anything, mock.Anything, mock.Anything).
    Run(func(args mock.Arguments) {
        msg := args.Get(2).(Message)
        t.Logf("mock received: %s", msg.Subject)
    }).Return(nil)
```

## Different returns per call

Stacked expectations consume in order — the classic retry test:

```go
m.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("timeout")).Once()
m.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
```

## Removing expectations

A registered expectation can be swapped out — reuse the *Call handle returned by `On`:

```go
call := m.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)
call.Unset()
m.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("fail"))
```

## Verification

```go
m.AssertExpectations(t)                                                        // all registered expectations were met
m.AssertCalled(t, "Send", mock.Anything, "buyer@example.com", mock.Anything)   // a specific call happened
m.AssertNotCalled(t, "BatchSend", mock.Anything, mock.Anything, mock.Anything) // a specific call did NOT happen
m.AssertNumberOfCalls(t, "Send", 2)                                            // exact call count
```

Call `AssertExpectations` at the end of every test — without it, unmet expectations pass silently.
