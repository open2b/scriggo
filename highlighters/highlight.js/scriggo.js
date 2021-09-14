/*
Language: Scriggo
Requires: xml.js
Author: Marco Gazerro <gazerro@open2b.com>
Contributors:
Description: Scriggo is the world's powerful template engine with Go as scripting language.
Website: https://scriggo.com
Category: template
*/

export default function(hljs) {

  var KEYWORDS = {
    keyword:
      'and break case chan const contains continue default defer else end extends ' +
      'if import in fallthrough for func go goto interface macro map not ' +
      'or range render return select show struct switch type using var ',
    type:
      'bool byte complex64 complex128 float32 float64 int int8 int16 ' +
      'int32 int64 rune string uint uint8 uint16 uint32 uint64 uintptr ' +
      'html markdown css js json',
    literal:
      'itea iota false nil true',
    builtin:
      'append cap close complex copy delete imag len make new panic print ' +
      'println real recover'
  };

  var CODE = [
    {
      className: 'string',
      variants: [
        hljs.APOS_STRING_MODE,
        hljs.QUOTE_STRING_MODE,
        { begin: '`', end: '`' },
      ]
    },
    {
      className: 'number',
      variants: [
        hljs.C_NUMBER_MODE
      ]
    },
    hljs.C_BLOCK_COMMENT_MODE
  ];

  return {
    name: 'Scriggo',
    subLanguage: 'xml',
    contains: [
      {
        className: 'show',
        begin: /\{\{/, end: /}}/,
        keywords: KEYWORDS,
        contains: CODE
      },
      {
        className: 'statement',
        begin: /\{%/, end: /%}/,
        keywords: KEYWORDS,
        contains: CODE
      },
      hljs.COMMENT(/\{#/, /#}/, { contains: [ 'self' ] })
    ]
  };

}
