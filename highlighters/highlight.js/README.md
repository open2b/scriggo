# highlight.js

This is the syntax highlighter for [highlight.js](https://highlightjs.org/).

## How to build

Here are the steps to build a *highlight.js* file that supports Scriggo.

### Requirements

- [node](https://nodejs.org/it/)

### Steps

1. Clone the repository of *highlight.js*:
   ```bash
   git clone https://github.com/highlightjs/highlight.js
   ```
2. Enter into it:
   ```bash
   cd highlight.js
   ```
3. Checkout the tag of the latest release (you can find it on the [releases
   page](https://github.com/highlightjs/highlight.js/releases)). For example, if the
   latest release is *11.2.0*:
   ```bash
   git checkout 11.2.0
   ```
4. Install the dependencies of *highlight.js* using *npm*:
   ```bash
   npm install
   ```
5. Copy the `scriggo.js` file from the repository of *Scriggo* to the repository of
   *highlight.js*:
   ```bash
   cp <path to Scriggo repo>/highlighters/highlight.js/scriggo.js src/languages
   ```
6. Build the syntax source:
   ```bash
   node tools/build.js scriggo
   ```
7. Copy the result in the destination directory:
   ```bash
   cp build/highlight.js <destination directory>
   ```
