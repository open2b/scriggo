
{% macro B(items []string) %}
{% show b %}
{% for item in items %}
* {{ item }}
{% end %}
{% end macro %}

{% macro b %}
## items
{% end macro %}
