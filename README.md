# Lint Struct Padding

# What's this?
You can read [this article](https://kushallabs.com/understanding-struct-padding-in-go-in-depth-guide-ed70c0432c63) to know more about performance improvement on golang structs.  
This linter helps you find (and fix) the ordering issues of your golang structs fields.

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
- [ ] Improve README file with a terminal GIF showing what it does
