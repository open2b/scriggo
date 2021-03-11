
{% macro B(items []string) %}
{{ b() }}
{% for item in items %}
* {{ item }}
{% end %}
{% end macro %}

{% macro b %}
## items
{% end macro %}
