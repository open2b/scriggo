// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"bytes"
	"scriggo/template"
	"testing"
)

func TestFullTemplate(t *testing.T) {
	r := template.DirReader("./full_template")
	for page, expectedOutput := range expectedPagesOutput {
		t.Run(page, func(t *testing.T) {
			templ, err := template.Load(page, r, nil, template.ContextHTML, nil)
			if err != nil {
				t.Fatal(err)
			}
			out := &bytes.Buffer{}
			err = templ.Render(out, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != expectedOutput {
				if testing.Verbose() {
					t.Fatalf("expecting:\n%q\ngot:\n%q", expectedOutput, out.String())
				} else {
					t.Fatalf("output is not what expected (use -v to get details)")
				}
			}
		})
	}
}

var expectedPagesOutput = map[string]string{
	"/index.html": "\n\n<!DOCTYPE html>\n<html class=\"-html\">\n<head itemscope itemtype=\"http://schema.org/WebSite\">\n  <title itemprop=\"name\"></title>\n  \n  \n  \n  <meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\">\n  <meta name=\"viewport\" content=\"initial-scale=1.0, width=device-width, user-scalable=no\">\n  \n</head>\n<body>\n\n<div class=\"body-content\">\n\n\n<header>\n\n  <div class=\"centered\">\n    <div class=\"search\">\n      <div id=\"search-panel\" class=\"panel\">\n        <div>\n          <input type=\"text\" name=\"keywords\" class=\"design-search-keywords\" placeholder=\"{Search...}\" size=\"15\">\n          <div class=\"search-button icon-search main-button\">\n          </div>\n          </form>\n        </div>\n      </div>\n    </div>\n\n    <div class=\"logo\">\n    </div>\n\n    <div class=\"cart\">\n      </a>\n      \n      <div class=\"content\">\n\n\n      </div>\n\n    </div>\n  </div>\n\n</header>\n\n\n\n<nav class=\"nav\" role=\"navigation\">\n\n  <div class=\"general\">\n\n      <div class=\"account\">\n        <div id=\"account-panel\" class=\"panel\">\n        </div>\n      </div>\n  </div>\n\n  <div class=\"bar\">\n    <a class=\"opener\" data-design-open=\"bar-panel\">\n      <span></span>\n      <span></span>\n      <span></span>\n      <span></span>\n    </a>\n\n    <div id=\"bar-panel\" class=\"panel\">\n\n      <ul class=\"main-items\">\n        <li class=\"megamenu has-panel\">\n          <div class=\"panel\" data-design-set-height=\"(max-width:1023px)\">\n            <div class=\"content\">\n\n              <div class=\"menu\">\n              </div>\n\n              <div class=\"menu\">\n              </div>\n\n              <div class=\"menu\">\n              </div>\n\n            </div>\n          </div>\n        </li>\n      </ul>\n\n    </div>\n  </div>\n\n</nav>\n\n\n\n\n  \n\n\n\n<div class=\"main\" role=\"main\">\n    \n\n</div>\n\n</div>\n\n\n<footer>\n  <div class=\"footer-content\">\n\n    <div class=\"upper\">\n  \n      <div class=\"footnote\">\n        \n      </div>\n\n      <div class=\"social-networks\">\n      </div>\n\n      <div class=\"menus\">\n      </div>\n\n    </div>\n    <div class=\"powered-by\"></div>\n\n  </div>\n</footer>\n\n\n<div id=\"design-overlay\" class=\"overlay\"></div>\n<div id=\"filters-overlay\" class=\"overlay\"></div>\n\n</body>\n</html>\n",

	"/login.html": "<!DOCTYPE html>\n<head>\n  <meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\">\n  <meta name=\"viewport\" content=\"initial-scale=1.0, width=device-width, user-scalable=no\">\n   \n</head>\n<body>\n\n<div class=\"body-content\">\n\n\n<header>\n\n  <div class=\"centered\">\n    <div class=\"search\">\n      <div id=\"search-panel\" class=\"panel\">\n        <div>\n          <input type=\"text\" name=\"keywords\" class=\"design-search-keywords\" placeholder=\"{Search...}\" size=\"15\">\n          <div class=\"search-button icon-search main-button\">\n          </div>\n          </form>\n        </div>\n      </div>\n    </div>\n\n    <div class=\"logo\">\n    </div>\n\n    <div class=\"cart\">\n      </a>\n      \n      <div class=\"content\">\n\n\n      </div>\n\n    </div>\n  </div>\n\n</header>\n\n\n\n<nav class=\"nav\" role=\"navigation\">\n\n  <div class=\"general\">\n\n      <div class=\"account\">\n        <div id=\"account-panel\" class=\"panel\">\n        </div>\n      </div>\n  </div>\n\n  <div class=\"bar\">\n    <a class=\"opener\" data-design-open=\"bar-panel\">\n      <span></span>\n      <span></span>\n      <span></span>\n      <span></span>\n    </a>\n\n    <div id=\"bar-panel\" class=\"panel\">\n\n      <ul class=\"main-items\">\n        <li class=\"megamenu has-panel\">\n          <div class=\"panel\" data-design-set-height=\"(max-width:1023px)\">\n            <div class=\"content\">\n\n              <div class=\"menu\">\n              </div>\n\n              <div class=\"menu\">\n              </div>\n\n              <div class=\"menu\">\n              </div>\n\n            </div>\n          </div>\n        </li>\n      </ul>\n\n    </div>\n  </div>\n\n</nav>\n\n\n<div class=\"main\" role=\"main\">\n    \n\n\n  <div class=\"login-button main-button\">\n  </div>\n\n  </form>\n\n\n</div>\n\n</div>\n\n\n<footer>\n  <div class=\"footer-content\">\n\n    <div class=\"upper\">\n  \n      <div class=\"footnote\">\n        \n      </div>\n\n      <div class=\"social-networks\">\n      </div>\n\n      <div class=\"menus\">\n      </div>\n\n    </div>\n    <div class=\"powered-by\"></div>\n\n  </div>\n</footer>\n\n\n<div id=\"design-overlay\" class=\"overlay\"></div>\n<div id=\"filters-overlay\" class=\"overlay\"></div>\n\n</body>\n</html>\n",
}
