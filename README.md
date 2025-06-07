# Lint Struct Padding

# Description
You can read [this article](https://kushallabs.com/understanding-struct-padding-in-go-in-depth-guide-ed70c0432c63) to know more about performance improvement on golang structs.  
This linter helps you find (and fix) the ordering issues of your golang structs fields.

# Installation
```bash
go install github.com/majidalaeinia/lintstructpadding@latest
```

```bash
sudo mv $(go env GOPATH)/bin/lintstructpadding /usr/bin
```

### Run the Linter

#### specific file
```bash
lintstructpadding /path/to/a/singlefile.go
```

#### complete source code
```bash
cd /path/to/your/source/code
lintstructpadding
```

### Fix Struct Padding
#### specific file
```bash
lintstructpadding --fix /path/to/a/singlefile.go
```

#### complete source code
```bash
cd /path/to/your/source/code
lintstructpadding --fix
```

### TODO
- [ ] Add tests
- [ ] Remove blank lines on struct fixing
- [ ] Improve README file with a terminal GIF showing what it does
- [ ] Improve installation

