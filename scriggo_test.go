// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"reflect"
	"strings"
	"testing"

	"github.com/open2b/scriggo/compiler"
)

func TestInitPackageLevelVariables(t *testing.T) {

	// Test no globals.
	globals := initPackageLevelVariables([]compiler.Global{})
	if globals != nil {
		t.Fatalf("expected nil, got %v", globals)
	}

	// Test zero value.
	global := compiler.Global{
		Pkg:  "p",
		Name: "a",
		Type: reflect.TypeOf(0),
	}
	globals = initPackageLevelVariables([]compiler.Global{global})
	g := globals[0]
	if g.Kind() != reflect.Int {
		t.Fatalf("unexpected kind %v", g.Kind())
	}
	if g.Interface() != 0 {
		t.Fatalf("unexpected %v, expecting 0", g.Interface())
	}

	// Test pointer value in globals.
	n := 1
	global = compiler.Global{
		Pkg:   "p",
		Name:  "a",
		Type:  reflect.TypeOf(n),
		Value: reflect.ValueOf(&n).Elem(),
	}
	globals = initPackageLevelVariables([]compiler.Global{global})
	n = 2
	g = globals[0]
	if g.Kind() != reflect.Int {
		t.Fatalf("unexpected kind %v", g.Kind())
	}
	iface := g.Interface()
	if iface != n {
		t.Fatalf("unexpected %v (type %T), expecting %d (type %T)", iface, iface, n, n)
	}

}

func TestIssue523(t *testing.T) {
	// See https://github.com/open2b/scriggo/issues/523.
	src := `package main

	import "fmt"

	func main() {
		fmt.Println("hello")
	}`
	_, _ = Build(strings.NewReader(src), nil)
}

func Test_LimitExceededError(t *testing.T) {
	src := `
	package main

func main() {
	var v1 int ; _ = v1
	var v2 int ; _ = v2
	var v3 int ; _ = v3
	var v4 int ; _ = v4
	var v5 int ; _ = v5
	var v6 int ; _ = v6
	var v7 int ; _ = v7
	var v8 int ; _ = v8
	var v9 int ; _ = v9
	var v10 int ; _ = v10
	var v11 int ; _ = v11
	var v12 int ; _ = v12
	var v13 int ; _ = v13
	var v14 int ; _ = v14
	var v15 int ; _ = v15
	var v16 int ; _ = v16
	var v17 int ; _ = v17
	var v18 int ; _ = v18
	var v19 int ; _ = v19
	var v20 int ; _ = v20
	var v21 int ; _ = v21
	var v22 int ; _ = v22
	var v23 int ; _ = v23
	var v24 int ; _ = v24
	var v25 int ; _ = v25
	var v26 int ; _ = v26
	var v27 int ; _ = v27
	var v28 int ; _ = v28
	var v29 int ; _ = v29
	var v30 int ; _ = v30
	var v31 int ; _ = v31
	var v32 int ; _ = v32
	var v33 int ; _ = v33
	var v34 int ; _ = v34
	var v35 int ; _ = v35
	var v36 int ; _ = v36
	var v37 int ; _ = v37
	var v38 int ; _ = v38
	var v39 int ; _ = v39
	var v40 int ; _ = v40
	var v41 int ; _ = v41
	var v42 int ; _ = v42
	var v43 int ; _ = v43
	var v44 int ; _ = v44
	var v45 int ; _ = v45
	var v46 int ; _ = v46
	var v47 int ; _ = v47
	var v48 int ; _ = v48
	var v49 int ; _ = v49
	var v50 int ; _ = v50
	var v51 int ; _ = v51
	var v52 int ; _ = v52
	var v53 int ; _ = v53
	var v54 int ; _ = v54
	var v55 int ; _ = v55
	var v56 int ; _ = v56
	var v57 int ; _ = v57
	var v58 int ; _ = v58
	var v59 int ; _ = v59
	var v60 int ; _ = v60
	var v61 int ; _ = v61
	var v62 int ; _ = v62
	var v63 int ; _ = v63
	var v64 int ; _ = v64
	var v65 int ; _ = v65
	var v66 int ; _ = v66
	var v67 int ; _ = v67
	var v68 int ; _ = v68
	var v69 int ; _ = v69
	var v70 int ; _ = v70
	var v71 int ; _ = v71
	var v72 int ; _ = v72
	var v73 int ; _ = v73
	var v74 int ; _ = v74
	var v75 int ; _ = v75
	var v76 int ; _ = v76
	var v77 int ; _ = v77
	var v78 int ; _ = v78
	var v79 int ; _ = v79
	var v80 int ; _ = v80
	var v81 int ; _ = v81
	var v82 int ; _ = v82
	var v83 int ; _ = v83
	var v84 int ; _ = v84
	var v85 int ; _ = v85
	var v86 int ; _ = v86
	var v87 int ; _ = v87
	var v88 int ; _ = v88
	var v89 int ; _ = v89
	var v90 int ; _ = v90
	var v91 int ; _ = v91
	var v92 int ; _ = v92
	var v93 int ; _ = v93
	var v94 int ; _ = v94
	var v95 int ; _ = v95
	var v96 int ; _ = v96
	var v97 int ; _ = v97
	var v98 int ; _ = v98
	var v99 int ; _ = v99
	var v100 int ; _ = v100
	var v101 int ; _ = v101
	var v102 int ; _ = v102
	var v103 int ; _ = v103
	var v104 int ; _ = v104
	var v105 int ; _ = v105
	var v106 int ; _ = v106
	var v107 int ; _ = v107
	var v108 int ; _ = v108
	var v109 int ; _ = v109
	var v110 int ; _ = v110
	var v111 int ; _ = v111
	var v112 int ; _ = v112
	var v113 int ; _ = v113
	var v114 int ; _ = v114
	var v115 int ; _ = v115
	var v116 int ; _ = v116
	var v117 int ; _ = v117
	var v118 int ; _ = v118
	var v119 int ; _ = v119
	var v120 int ; _ = v120
	var v121 int ; _ = v121
	var v122 int ; _ = v122
	var v123 int ; _ = v123
	var v124 int ; _ = v124
	var v125 int ; _ = v125
	var v126 int ; _ = v126
	var v127 int ; _ = v127
	var v128 int ; _ = v128
	var v129 int ; _ = v129
	var v130 int ; _ = v130
	var v131 int ; _ = v131
	var v132 int ; _ = v132
	var v133 int ; _ = v133
	var v134 int ; _ = v134
	var v135 int ; _ = v135
	var v136 int ; _ = v136
	var v137 int ; _ = v137
	var v138 int ; _ = v138
	var v139 int ; _ = v139
	var v140 int ; _ = v140
	var v141 int ; _ = v141
	var v142 int ; _ = v142
	var v143 int ; _ = v143
	var v144 int ; _ = v144
	var v145 int ; _ = v145
	var v146 int ; _ = v146
	var v147 int ; _ = v147
	var v148 int ; _ = v148
	var v149 int ; _ = v149
	var v150 int ; _ = v150
	var v151 int ; _ = v151
	var v152 int ; _ = v152
	var v153 int ; _ = v153
	var v154 int ; _ = v154
	var v155 int ; _ = v155
	var v156 int ; _ = v156
	var v157 int ; _ = v157
	var v158 int ; _ = v158
	var v159 int ; _ = v159
	var v160 int ; _ = v160
	var v161 int ; _ = v161
	var v162 int ; _ = v162
	var v163 int ; _ = v163
	var v164 int ; _ = v164
	var v165 int ; _ = v165
	var v166 int ; _ = v166
	var v167 int ; _ = v167
	var v168 int ; _ = v168
	var v169 int ; _ = v169
	var v170 int ; _ = v170
	var v171 int ; _ = v171
	var v172 int ; _ = v172
	var v173 int ; _ = v173
	var v174 int ; _ = v174
	var v175 int ; _ = v175
	var v176 int ; _ = v176
	var v177 int ; _ = v177
	var v178 int ; _ = v178
	var v179 int ; _ = v179
	var v180 int ; _ = v180
	var v181 int ; _ = v181
	var v182 int ; _ = v182
	var v183 int ; _ = v183
	var v184 int ; _ = v184
	var v185 int ; _ = v185
	var v186 int ; _ = v186
	var v187 int ; _ = v187
	var v188 int ; _ = v188
	var v189 int ; _ = v189
	var v190 int ; _ = v190
	var v191 int ; _ = v191
	var v192 int ; _ = v192
	var v193 int ; _ = v193
	var v194 int ; _ = v194
	var v195 int ; _ = v195
	var v196 int ; _ = v196
	var v197 int ; _ = v197
	var v198 int ; _ = v198
	var v199 int ; _ = v199
	var v200 int ; _ = v200
	var v201 int ; _ = v201
	var v202 int ; _ = v202
	var v203 int ; _ = v203
	var v204 int ; _ = v204
	var v205 int ; _ = v205
	var v206 int ; _ = v206
	var v207 int ; _ = v207
	var v208 int ; _ = v208
	var v209 int ; _ = v209
	var v210 int ; _ = v210
	var v211 int ; _ = v211
	var v212 int ; _ = v212
	var v213 int ; _ = v213
	var v214 int ; _ = v214
	var v215 int ; _ = v215
	var v216 int ; _ = v216
	var v217 int ; _ = v217
	var v218 int ; _ = v218
	var v219 int ; _ = v219
	var v220 int ; _ = v220
	var v221 int ; _ = v221
	var v222 int ; _ = v222
	var v223 int ; _ = v223
	var v224 int ; _ = v224
	var v225 int ; _ = v225
	var v226 int ; _ = v226
	var v227 int ; _ = v227
	var v228 int ; _ = v228
	var v229 int ; _ = v229
	var v230 int ; _ = v230
	var v231 int ; _ = v231
	var v232 int ; _ = v232
	var v233 int ; _ = v233
	var v234 int ; _ = v234
	var v235 int ; _ = v235
	var v236 int ; _ = v236
	var v237 int ; _ = v237
	var v238 int ; _ = v238
	var v239 int ; _ = v239
	var v240 int ; _ = v240
	var v241 int ; _ = v241
	var v242 int ; _ = v242
	var v243 int ; _ = v243
	var v244 int ; _ = v244
	var v245 int ; _ = v245
	var v246 int ; _ = v246
	var v247 int ; _ = v247
	var v248 int ; _ = v248
	var v249 int ; _ = v249
	var v250 int ; _ = v250
	var v251 int ; _ = v251
	var v252 int ; _ = v252
	var v253 int ; _ = v253
	var v254 int ; _ = v254
	var v255 int ; _ = v255
	var v256 int ; _ = v256
	var v257 int ; _ = v257
	var v258 int ; _ = v258
	var v259 int ; _ = v259
	var v260 int ; _ = v260
	var v261 int ; _ = v261
	var v262 int ; _ = v262
	var v263 int ; _ = v263
	var v264 int ; _ = v264
	var v265 int ; _ = v265
	var v266 int ; _ = v266
	var v267 int ; _ = v267
	var v268 int ; _ = v268
	var v269 int ; _ = v269
	var v270 int ; _ = v270
	var v271 int ; _ = v271
	var v272 int ; _ = v272
	var v273 int ; _ = v273
	var v274 int ; _ = v274
	var v275 int ; _ = v275
	var v276 int ; _ = v276
	var v277 int ; _ = v277
	var v278 int ; _ = v278
	var v279 int ; _ = v279
	var v280 int ; _ = v280
	var v281 int ; _ = v281
	var v282 int ; _ = v282
	var v283 int ; _ = v283
	var v284 int ; _ = v284
	var v285 int ; _ = v285
	var v286 int ; _ = v286
	var v287 int ; _ = v287
	var v288 int ; _ = v288
	var v289 int ; _ = v289
	var v290 int ; _ = v290
	var v291 int ; _ = v291
	var v292 int ; _ = v292
	var v293 int ; _ = v293
	var v294 int ; _ = v294
	var v295 int ; _ = v295
	var v296 int ; _ = v296
	var v297 int ; _ = v297
	var v298 int ; _ = v298
	var v299 int ; _ = v299
	var v300 int ; _ = v300
}
	`
	_, err := Build(strings.NewReader(src), nil)
	if err == nil {
		t.Fatal("Expectend a LimitExceededError, got nothing")
	} else {
		if IsLimitExceeded(err) {
			// Test passed.
		} else {
			t.Fatalf("Expected a LimitExceededError, got %q (of type %T)", err, err)
		}
	}
}
