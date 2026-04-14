# Grove Cloud — Email Templates Example

Transactional email templates for SaaS: order confirmations, password resets, plan changes, and usage alerts.

## Quick Start

```bash
go run ./examples/email/
# Opens preview UI on http://localhost:8082
```

Click any template to preview with different user scenarios.

## What It Demonstrates

### Core Grove Features

- ✅ **Email-safe HTML** — Table layouts, inline styles, MSO compatibility (does **not** use the asset pipeline — email clients require inline styles, so `pkg/grove/assets` is not wired in)
- ✅ **Component helpers** — Reusable Button, Divider, Spacer, Heading, UsageBar
- ✅ **Preheaders** — `{% #hoist "preheader" %}` for email client subject line
- ✅ **Captured blocks** — `{% #capture %}` for multi-use template sections
- ✅ **Conditional rendering** — User-specific messaging, optional sections
- ✅ **Cross-example integration** — Order confirmations from Coldfront Supply Co.

### Production Email Patterns

- ✅ **Responsive width** — Fixed 600px max for consistency across clients
- ✅ **Fallback fonts** — System sans-serif stack with serif fallback
- ✅ **Button rendering** — Tables + MSO conditionals for Outlook support
- ✅ **Text alternatives** — Alt text for images, plain-text fallbacks
- ✅ **Unsubscribe links** — Footer with preference management

## File Organization

```
email/
├── main.go                           # Preview server, template registry
├── templates/
│   ├── base-email.grov               # <BaseEmail> layout with preheader slot
│   ├── index.grov                    # Preview/test landing page
│   ├── order-confirmation.grov       # Order receipt (uses Coldfront data)
│   ├── password-reset.grov           # Password reset flow
│   ├── plan-change.grov              # Plan upgrade/downgrade notification
│   ├── usage-alert.grov              # Usage % alert (API quota warning)
│   ├── welcome.grov                  # Welcome on-boarding email
│   ├── helpers.grov                  # Email component macros
│   └── README.md                     # This file
└── README.md                         # This file
```

## How It Works

### Email Templates

Each template extends `<BaseEmail>` and fills the `body` slot:

```grov
{% import BaseEmail from "base-email" %}

<BaseEmail>
  {% #fill "title" %}Your Email Subject{% /fill %}
  
  {% #hoist "preheader" %}Preview text shown in inbox...{% /hoist %}
  
  {% #fill "body" %}
    <p>Hi {% user.name %},</p>
    <!-- Email content -->
  {% /fill %}
</BaseEmail>
```

### Helper Components

`helpers.grov` provides reusable email patterns:

#### `<Button>`
MSO-safe CTA button using table layout:
```grov
<H.Button text="Reset Password" href="https://..." color="#2E6740" />
```

#### `<Divider>`
Horizontal line:
```grov
<H.Divider />
```

#### `<Spacer>`
Fixed-height spacing (email clients ignore CSS margins):
```grov
<H.Spacer height="24" />
```

#### `<Heading>`
Styled `<h2>` for email:
```grov
<H.Heading text="Order Summary" />
```

#### `<UsageBar>`
Progress bar for quota/usage display:
```grov
<H.UsageBar pct={usage_percent} />  <!-- Auto-colors: green/amber/red -->
```

### Preheader Optimization

The preheader text appears next to the subject in email clients:

```grov
{% #hoist "preheader" %}
  Order #{% order.id %} confirmed. Total: {% order.total | currency %}.
{% /hoist %}
```

Without a preheader, email clients show the first 50 characters of the body (usually `<p>Hi John,</p>`).

### Data Model

**User:**
```go
type User struct {
    ID       string
    Email    string
    Name     string
    Plan     string      // e.g., "Pro", "Enterprise"
    Company  string
}
```

**Order (Coldfront integration):**
```go
type Order struct {
    ID    string
    Items []OrderItem
    Total float64
}

type OrderItem struct {
    Name     string
    Quantity int
    Price    float64
}
```

## Template Breakdown

### Welcome (`welcome.grov`)
On-boarding email with:
- Warm greeting
- Account activation link
- Feature highlights
- Support contact

### Order Confirmation (`order-confirmation.grov`)
Transaction receipt with:
- Order ID and date
- Line-item table (product, qty, price)
- Total + tax breakdown
- Order tracking link
- Receipt delivery confirmation

**Cross-example note:** This uses Coldfront Supply Co. as the merchant, linking examples together.

### Password Reset (`password-reset.grov`)
Secure password change with:
- User name
- Reset link (time-sensitive)
- Validity period
- Security note ("ignore if you didn't request this")

### Plan Change (`plan-change.grov`)
Plan upgrade/downgrade with:
- Old vs. new plan
- Billing change
- Effective date
- Feature comparison
- Cancellation link

### Usage Alert (`usage-alert.grov`)
Quota warning with:
- Current usage % (with colored bar)
- Quota limits
- Upgrade CTA
- Usage breakdown by resource

## Styling

Email styles are **inline** (no external CSS). Common patterns:

```html
<table cellpadding="0" cellspacing="0" border="0" style="...">
  <tr>
    <td style="padding: 16px; text-align: left; color: #333;">
      Content here
    </td>
  </tr>
</table>
```

### MSO Conditionals

Outlook requires special handling for fallbacks:

```html
<!--[if mso]>
  <table cellpadding="0" cellspacing="0" border="0">
    <tr><td style="...">...</td></tr>
  </table>
<![endif]-->
```

(The `<Button>` helper includes these automatically.)

## Editing Templates

### Change email header color
In `base-email.grov`:
```html
<div class="header" style="background-color: #YOUR_HEX;">
```

### Add a new email type
1. Create `templates/your-template.grov`
2. Import `BaseEmail` and helpers
3. Fill `title`, `preheader`, and `body` slots
4. Register in `main.go`:
```go
templates["your-template"] = engine.MustCompile("your-template")
```
5. Add to preview UI in `index.grov`

### Customize user data
In `main.go`, edit the `users` fixture:
```go
{ID: "3", Name: "Sarah", Email: "...", Plan: "Enterprise"},
```

## Testing Email

### Preview Server

The preview server (`main.go`) renders templates with different user scenarios:
- Visit `/preview/order-confirmation?user=1` to see user #1's order email
- View source with "View Source" link

### Email Client Testing

Copy the rendered HTML and test in:
- **Litmus** — Tests across 70+ email clients
- **Email on Acid** — CSS rendering validation
- **MJML** — Modern email framework (for comparison)

### Common Issues

**Tables don't line up:** Email clients have different table spacing. Use `cellpadding="0" cellspacing="0"`.

**Colors stripped:** Some clients remove `style` attributes. Use `bgcolor` attribute as fallback.

**Images not showing:** Always include `alt` text and provide text fallback.

**Links clicked multiple times:** Use `href` on `<a>`, not `style="cursor: pointer"`.

## Accessibility in Email

✅ **Alt text on images** — Critical for screen readers  
✅ **Semantic headings** — Use `<h1>`, `<h2>`, not styled divs  
✅ **Link text** — "Click here" is bad, "Reset your password" is good  
✅ **Color contrast** — Text on background must meet WCAG AA  
✅ **Plain text fallback** — Multipart MIME with text/plain part  

## Performance

- Templates compile once at startup
- Rendering ~5ms per email
- No external API calls (all data local)

## Common Edits

### Change sender name
In `main.go`, edit template data:
```go
"sender_name": "Coldfront Support",
```

### Add company logo
In `base-email.grov`:
```html
<img src="https://cdn.example.com/logo.png" alt="Logo" width="200">
```

### Change brand color
Global token in template headers (all email templates):
Find and replace `#2E6740` → your color

### Add custom footer links
In `base-email.grov` footer slot:
```grov
<p>
  <a href="https://...">Unsubscribe</a> &middot;
  <a href="https://...">Preferences</a> &middot;
  <a href="https://...">Help</a>
</p>
```

## Debugging

### View rendered HTML
Click "View Source" on preview page — copies HTML to clipboard.

### Check Outlook rendering
Many email designers struggle with Outlook. Use MSO conditionals and avoid:
- CSS floats (use tables)
- Percentage widths (use fixed pixels)
- Background images (background-color only)

### Validate MJML/Pug templates
If converting to MJML, [use their validator](https://mjml.io/validate).

---

See `/examples/README.md` for context on other examples and shared design system.
