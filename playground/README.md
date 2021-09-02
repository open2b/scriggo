# How to compile

## Linux

```
export GOOS=js
export GOARCH=wasm
scriggo import -v -o packages.go
go build -tags osusergo,netgo -trimpath -o scriggo.wasm
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .
```

## Windows

```
SET GOOS=js
SET GOARCH=wasm
scriggo import -v -o packages.go
go build -tags osusergo,netgo -trimpath -o scriggo.wasm
copy "C:\Program Files\Go\misc\wasm\wasm_exec.js" .
```

Than serve the files `index.html`, `playground.js`, `scriggo.wasm` and `wasm_exec.js`
with a web server.
