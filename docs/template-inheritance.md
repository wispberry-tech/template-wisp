# Layouts

Grove uses component composition for layouts — there is no separate template inheritance system. A layout is simply a component with named slots.

See [Components — Layouts via Components](components.md#layouts-via-components) for the full documentation.

## Quick Example

Define a layout as a component:

```html
{# base.html #}
<Component name="Base">
  <!DOCTYPE html>
  <html>
  <head>
    <title><Slot name="title">My Site</Slot></title>
  </head>
  <body>
    <main><Slot name="content" /></main>
    <footer><Slot name="footer">&copy; 2026</Slot></footer>
  </body>
  </html>
</Component>
```

Pages import and fill slots:

```html
{# home.html #}
<Import src="base" name="Base" />
<Base>
  <Fill slot="title">Home — My Site</Fill>
  <Fill slot="content">
    <h1>Welcome</h1>
  </Fill>
</Base>
```

## Multi-Level Layouts

Layouts can compose other layouts:

```html
{# section.html #}
<Import src="base" name="Base" />
<Component name="Section">
  <Base>
    <Fill slot="content">
      <div class="section">
        <Slot name="inner">section default</Slot>
      </div>
    </Fill>
  </Base>
</Component>
```

```html
{# page.html #}
<Import src="section" name="Section" />
<Section>
  <Fill slot="inner">page content</Fill>
</Section>
```

Rendering `page.html` produces:

```html
<!DOCTYPE html>
<html>
<head>
  <title>My Site</title>
</head>
<body>
  <main><div class="section">
    page content
  </div></main>
  <footer>&copy; 2026</footer>
</body>
</html>
```
