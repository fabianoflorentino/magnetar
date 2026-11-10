# Variables, Booleans, Receivers & Acronyms

## Variable names

Length tracks scope: short names inside a small scope, descriptive names at package scale.

```go
for i, v := range items {            // 3-line scope: i and v are exact
    result = append(result, v.Name)
}

userCount := len(users)              // medium scope

var defaultHTTPTransport = &http.Transport{   // package scope: explicit
    MaxIdleConns: 100,
}
```

Conventional single-letter names:

| Name | Meaning |
| --- | --- |
| `i`, `j`, `k` | Loop indices |
| `n` | Count or length |
| `v` | Value in a `range` loop |
| `k` | Key in a map `range` |
| `r` | `io.Reader` |
| `w` | `io.Writer` |
| `b` | `[]byte` / buffer |
| `s` | String |
| `t` | `*testing.T` |
| `ctx` | `context.Context` |
| `err` | Error |

### No type in the name

The name describes what the value *is*, not its type:

```go
users := getUsers()    // good
count := len(items)

userSlice := getUsers()   // bad — the type is already in the declaration
nameString := "hello"
```

### No repetition with context

Omit what the enclosing function, receiver, or type already supplies:

```go
func (u *UserService) Create(name string) error  // receiver says "user"

// Creates a redundant noun
func (u *UserService) CreateUser(userName string) error
```

### One concept, one name

The same domain concept keeps the same identifier everywhere. `user` today must still be `user` tomorrow — not `account`, `person`, or `u` by mood:

```go
func CreateUser(user *User) error { ... }
func UpdateUser(user *User) error { ... }
func DeleteUser(userID string) error { ... }

// Same concept, scattered names
func UpdateAccount(acct *User) error { ... }
func RemovePerson(id string) error { ... }
```

Applies to variables, parameters, fields, and `userID`-style tokens alike — never `uid` in one place and `userId` in another.

### Parameters

A parameter is documentation at the call site. When the type is self-describing, keep it short; when the type is ambiguous (`int64`, `string`), name the meaning:

```go
func AfterFunc(d Duration, f func()) *Timer   // Duration is descriptive
func Escape(w io.Writer, s []byte)

func Unix(sec, nsec int64) Time                // int64 says nothing — name it
func HasPrefix(s, prefix string) bool          // two strings need roles

func Unix(a, b int64) Time                     // what are a and b?
```

## Booleans

Booleans name a question that answers true/false. Prefix variables and struct fields with `is`, `has`, `can`, `allow`, or `should`:

```go
type Client struct {
    isConnected   bool  // "client is connected"
    hasPermission bool  // "client has permission"
}

isReady := true
hasPermission := user.CanEdit(doc)

// Bare nouns read like values, not questions
connected bool   // looks like a connection object
permission bool  // a noun, not a predicate
```

The prefix survives export: an unexported `isConnected` field is exposed as `IsConnected() bool`.

## Receivers

Receivers are 1–2 letter abbreviations of the type name — and stay the same across every method of that type:

```go
func (s *Server) Start() error      { ... }
func (s *Server) Stop() error       { ... }

func (server *Server) Start() error { ... }   // too long
func (s *Server) Start() error      { ... }   // then a different receiver below
func (srv *Server) Stop() error     { ... }   // is churn

func (this *Server) Handle(r *Request) { ... }  // never this/self
```

## Acronyms and initialisms

Acronyms are all caps or all lower — never mixed — so they stay readable inside MixedCaps:

```go
URL            // all caps
url            // all lower
HTTPServer     // HTTP all caps
xmlParser      // xml all lower
userID         // ID all caps

Url            // bad
HttpServer     // bad
userId          // bad
```

Exporting a lowercase acronym capitalizes the whole token: `url` → `URL`, `grpc` → `GRPC`, `ios` → `IOS`.