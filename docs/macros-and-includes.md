# Components Replace Macros & Includes

In Grove's HTML-centric syntax, macros, includes, and imports are all replaced by the unified `<Component>` and `<Import>` system. See [Components](components.md) for the full documentation.

## Migration from Legacy Syntax

### Macros → Components

**Before (legacy):**
```
{% macro user_card(name, role="member") %}
  <div class="card">
    <strong>{{ name }}</strong>
    <span class="role">{{ role }}</span>
  </div>
{% endmacro %}

{{ user_card("Alice", "admin") }}
```

**After:**
```html
{# user-card.html #}
<Component name="UserCard" name role="member">
  <div class="card">
    <strong>{% name %}</strong>
    <span class="role">{% role %}</span>
  </div>
</Component>

{# page.html #}
<Import src="user-card" name="UserCard" />
<UserCard name="Alice" role="admin" />
```

### Includes → Import + Component

**Before (legacy):**
```
{% include "partials/nav.grov" %}
{% render "partials/card.grov" title="Widget" %}
```

**After:**
```html
<Import src="partials/nav" name="Nav" />
<Import src="partials/card" name="Card" />

<Nav />
<Card title="Widget" />
```

All components have isolated scope — there is no shared-scope include. Pass data explicitly via props.

### Import namespace → Wildcard import

**Before (legacy):**
```
{% import "macros/ui.grov" as ui %}
{{ ui.user_card("Alice") }}
```

**After:**
```html
<Import src="macros/ui" name="*" as="UI" />
<UI.UserCard name="Alice" />
```

### call/caller → Slots

**Before (legacy):**
```
{% macro card(title) %}
  <div class="card">
    <h2>{{ title }}</h2>
    {{ caller() }}
  </div>
{% endmacro %}

{% call card("Orders") %}
  <p>3 pending orders</p>
{% endcall %}
```

**After:**
```html
{# card.html #}
<Component name="Card" title>
  <div class="card">
    <h2>{% title %}</h2>
    <Slot />
  </div>
</Component>

{# page.html #}
<Import src="card" name="Card" />
<Card title="Orders">
  <p>3 pending orders</p>
</Card>
```
