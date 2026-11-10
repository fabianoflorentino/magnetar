# Filesystem Security Rules

Filesystem bugs read files the user should not see, write where they should not write, and consume unbounded disk. Rules:

1. User-controlled paths must be confined to an allowed root.
2. Prefer `os.Root` for scoped access on Go 1.24+.
3. Zip extraction must reject path traversal (ZipSlip).
4. Temporary files must use `os.CreateTemp` - never predictable names.
5. Secrets get `0600`; directories get `0750` at most.

## Directory traversal - High

Path components like `../../etc/passwd` escape an intended directory. Naive string checks are unsafe - `filepath.Clean` with `HasPrefix` is fooled by `./` tricks and sibling prefixes.

**Bad** - a bare join trusts the input:

```go
p := filepath.Join("/var/www", filename) // filename may contain ..
http.ServeFile(w, r, p)
```

**Good (Go 1.24+) - `os.Root` confines all operations:**

```go
root, err := os.OpenRoot("/var/www")
if err != nil {
    return err
}
defer root.Close()
f, err := root.Open(filename) // cannot leave the root
```

`os.Root` enforces confinement at the OS level: traversal and symlinks escaping the root are rejected across `Open`, `Create`, `Stat`, and friends. It is not a full sandbox - it does not block bind mounts, device files, or `/proc`-style access - so for uploads and archive extraction still reject special files and pick a root without attacker-controlled mounts.

**Good (pre-1.24 fallback):**

```go
func safeJoin(baseDir, userPath string) (string, error) {
    if userPath == "" || filepath.IsAbs(userPath) || !filepath.IsLocal(userPath) {
        return "", errors.New("invalid relative path")
    }
    full := filepath.Join(baseDir, userPath)
    rel, err := filepath.Rel(baseDir, full)
    if err != nil {
        return "", fmt.Errorf("checking path: %w", err)
    }
    if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
        return "", errors.New("path escapes base directory")
    }
    return full, nil
}
```

This lexical fallback is not symlink-resistant; upgrade to `os.Root` when possible.

## Zip archive path traversal - High

A crafted archive can write outside the extraction directory. Validate every entry name before opening a file for it.

**Good (Go 1.24+):**

```go
root, err := os.OpenRoot(dest)
if err != nil {
    return err
}
defer root.Close()
for _, f := range reader.File {
    out, err := root.OpenFile(f.Name, os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err // escapes the root
    }
    // copy contents, then close
}
```

**Good (pre-1.24):** reject non-local names with `filepath.IsLocal` and run the `safeJoin` check above before any write.

## Decompression bomb - Medium

A few hundred bytes of gzip can expand to gigabytes. `io.Copy` has no opinion about size; wrap the reader with a hard limit and let the error propagate rather than pretending `io.EOF`:

```go
const maxDecompressedSize = 100 * 1024 * 1024

var errSizeLimit = errors.New("decompressed size limit exceeded")

type limitedReader struct {
    r    io.Reader
    read int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
    if l.read >= maxDecompressedSize {
        return 0, errSizeLimit // not io.EOF - io.Copy would treat EOF as success
    }
    n, err := l.r.Read(p)
    l.read += int64(n)
    return n, err
}
```

The same cap applies to PDFs, images, and any other format that decompresses or rasterizes on read.

## Temporary files - Medium

Predictable temp names are a symlink and TOCTOU lottery. `os.CreateTemp` picks a random name, and you should clamp permissions afterward:

```go
f, err := os.CreateTemp("", "myapp-*")
if err != nil {
    return err
}
defer os.Remove(f.Name())
if err := f.Chmod(0600); err != nil { ... }
```

## File and directory permissions - Medium

Open files with the least permissive mode that works: `0600` for secrets and configs, `0640` for group-shared logs, never world-writable `0666`/`0777`. Directories created with `os.MkdirAll(..., 0750)` stay group-only instead of opening the tree to everyone. When in doubt, start restrictive and widen deliberately.

## Tainted file read - High

Any read keyed off input - a filename, an ID, a report date - must be scoped before touching the filesystem:

```go
// Go 1.24+
root, err := os.OpenRoot("/var/www/public")
if err != nil {
    return nil, err
}
defer root.Close()
f, err := root.Open(filename)
if err != nil {
    return nil, err
}
defer f.Close()
return io.ReadAll(f)
```

## CWE references

- CWE-22 - path traversal
- CWE-409 - zip bomb decompression
- CWE-379 - insecure temp file creation
- CWE-732 - incorrect file permissions
