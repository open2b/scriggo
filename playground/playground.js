// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

(function() {

    var source;

    function refreshLineNumbers() {
        var i = 1;
        var nn = document.getElementById("LineNumbers");
        var last = nn.lastElementChild;
        if ( last != null ) {
            i = parseInt(last.textContent) + 1;
        }
        console.log(nn.offsetHeight, source.offsetHeight);
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
        ).then(bytes =>
            WebAssembly.instantiate(bytes, importObject)
        ).then(results =>
            results.instance
        );
    }

    var go = new Go();
    var mod = fetchAndInstantiate("scriggo.wasm", go.importObject);

    window.onload = function () {

        source = document.getElementById("Source");

        refreshLineNumbers();
        source.addEventListener('scroll', refreshLineNumbers);
        window.addEventListener('resize', refreshLineNumbers);

        mod.then(function (instance) {
            go.run(instance);
            var button = document.getElementById("Execute");
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
            button.addEventListener("click", function () {
                output.innerHTML = "";
                Scriggo.load(source.value, function (program, error) {
                    if (error != null) {
                        global.fs.writeSync(2, error);
                        return;
                    }
                    error = program.run();
                    program.release();
                    if (error != null) {
                        global.fs.writeSync(2, error);
                    }
                });
            });
            source.addEventListener("keyup", function () {
                output.innerHTML = "";
                Scriggo.load(source.value, function (program, error) {
                    if (error != null) {
                        global.fs.writeSync(2, error);
                        return;
                    }
                    program.release();
                });
            });
        });

    };

})();

