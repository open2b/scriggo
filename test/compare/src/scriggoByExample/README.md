To generate examples from GoByExamples, run the command below in https://github.com/mmcgrana/gobyexample repository root after cloning it:

```bash
mkdir -p gofiles && rm -f gofiles/* && mkdir -p out && rm -f out/* && cp examples/*/*.go gofiles && cd gofiles && for f in *.go; do sed "1i\n" $f > ../out/$f; done
```

The `out` dir will be populated with `.go` files valid for testing.