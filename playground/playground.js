// Copyright 2020 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

(function() {

    var source;
    var program;

    function refreshLineNumbers() {
        var i = 1;
        var nn = document.getElementById("LineNumbers");
        var last = nn.lastElementChild;
        if ( last != null ) {
            i = parseInt(last.textContent) + 1;
        }
        while ( nn.offsetHeight < source.offsetHeight + source.scrollTop ) {
            var n = document.createElement("div");
            n.textContent = i;
            nn.appendChild(n);
            i++
        }
        nn.style.marginTop = (-source.scrollTop) + "px";
    }

    function fetchAndInstantiate(url, importObject) {
        return fetch(url).then(response =>
            response.arrayBuffer()
        ).then(bytes => {
            document.getElementById("Output").innerHTML = "";
            return WebAssembly.instantiate(bytes, importObject)
        }).then(results =>
            results.instance
        );
    }

    var go = new Go();
    var mod = fetchAndInstantiate("scriggo.wasm", go.importObject);

    window.onload = function () {

        var body = document.getElementsByTagName("body")[0];
        var bytecode = document.getElementById("ByteCode");
        var lineNumbers = document.getElementById("LineNumbers");
        source = document.getElementById("Source");

        refreshLineNumbers();
        source.addEventListener("scroll", refreshLineNumbers);
        window.addEventListener("resize", refreshLineNumbers);

        function buildProgram() {
            Scriggo.build(source.value, function (prog, error) {
                if (program != null) {
                    program.release();
                }
                var line = 0;
                if (error != null) {
                    var m = error.match(/:(\d+):/);
                    line = m == null ? 0 : parseInt(m[1]);
                }
                for ( var i = 0; i < lineNumbers.children.length; i++ ) {
                    lineNumbers.children[i].className = i+1 === line ? "error" : "";
                }
                if (error != null) {
                    program = null;
                    global.fs.writeSync(2, error);
                    return;
                }
                program = prog;
                if ( body.className === "disassembled" ) {
                    bytecode.textContent = program.disassemble();
                }
            });
        }

        mod.then(function (instance) {
            go.run(instance);
            var run = document.getElementById("Execute");
            var disassemble = document.getElementById("Disassemble");
            var output = document.getElementById("Output");
            const decoder = new TextDecoder("utf-8");
            global.fs.writeSync = function (fd, buf) {
                var str = typeof buf == "string" ? buf : decoder.decode(buf);
                var span = document.createElement("span");
                span.className = fd === 1 ? "stdout" : "stderr";
                span.textContent = str;
                output.appendChild(span);
                return str.length;
            };
            run.addEventListener("click", function () {
                if ( program == null ) return;
                output.innerHTML = "";
                program.run(function(error) {
                    if (error != null) {
                        global.fs.writeSync(2, error);
                    }
                });
            });
            disassemble.addEventListener("click", function () {
                if ( body.className === "disassembled" ) {
                    body.className = "";
                    bytecode.textContent = "";
                    return;
                }
                body.className = "disassembled";
                if ( program == null ) return;
                bytecode.textContent = program.disassemble();
                body.className = "disassembled";
            });
            source.addEventListener("keyup", function () {
                output.innerHTML = "";
                buildProgram();

            });
            buildProgram();
        });

    };

})();

