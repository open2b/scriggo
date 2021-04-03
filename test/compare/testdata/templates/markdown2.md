{# render #}

<a href="{{ `"` }}">

<script>var v = {{ "a" }};</script>

<style>div { background: url(data:image/png;base64,{{ []byte("\x12\x34\x89") }}) }</style>

<![CDATA[ {{ ".5" }} ]]>

    {{ "*** spaces code block ***" }}
	{{ "*** tab code block ***" }}

\{{ 1 }\}

\<a b="{{ `"` }}"></a>

{{ html(`<c d="#d">#e</c>`) }}

{{ html(`#a <!-- #a --> <![CDATA[ #a <b>#b</b> ]]> #a`) }}
