# highlight.js

This is the syntax highlighter for [highlight.js](https://highlightjs.org/).

## How to build

Requires [node](https://nodejs.org/it/).

### Build

1. `git clone https://github.com/highlightjs/highlight.js`
2. `cd highlight.js`
3. `cp <path of Scriggo>/scriggo/highlighters/highlight.js/scriggo.js src/languages/`
4. `node tools/build.js -n scriggo`
5. `cp build/highlight.js <destination directory>`
