# Lint Struct Padding

# Installation
```bash
git clone git@github.com:majidalaeinia/lintstructpadding.git
```

```bash
cd lintstructpadding
go build
```

### Run the Linter
```bash
./lintstructpadding ./examples/fileone.go
```

### Fix a Sample File
```bash
./lintstructpadding --fix ./examples/fileone.go
```

### Run the Linter in the Current Directory
```bash
./lintstructpadding
```

### Fix All .go Files in the Current Directory
```bash
./lintstructpadding --fix
```

### TODO
- [ ] Add tests
- [ ] Remove blank lines on struct fixing
