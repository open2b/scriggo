// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"strings"
	"testing"
)

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
}
	`
	_, err := CompileProgram(strings.NewReader(src), nil, Options{})
	if err != nil {
		if _, ok := err.(*LimitExceededError); ok {
			// Test passed.
		} else {
			t.Fatalf("Expected a LimitExceededError, got %q (of type %T)", err, err)
		}
	}
}
