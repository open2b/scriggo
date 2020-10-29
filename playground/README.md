# How to compile

## Linux

```
export GOOS=js
export GOARCH=wasm
scriggo embed > packages.go
go build -tags osusergo,netgo -trimpath -o scriggo.wasm
```

## Windows

```
SET GOOS=js
SET GOARCH=wasm
scriggo embed > packages.go
go build -tags osusergo,netgo -trimpath -o scriggo.wasm
```

Than serve the files `index.html`, `playground.js`, `scriggo.wasm` and `wasm_exec.js`
with a web server.
