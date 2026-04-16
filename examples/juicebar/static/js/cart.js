/*
 * Juicebar cart — client-side only, stored in localStorage.
 *
 * Intentionally small and dependency-free. The server SSRs static content;
 * this file hydrates the cart UI and keeps the nav badge accurate.
 *
 * Key: juicebar:cart
 * Value: JSON-encoded [{ handle, size, qty }]
 */
(function () {
  'use strict';

  var STORAGE_KEY = 'juicebar:cart';
  var PRODUCTS_URL = '/static/data/products.json';

  var productsPromise = null;
  function loadProducts() {
    if (!productsPromise) {
      productsPromise = fetch(PRODUCTS_URL).then(function (r) { return r.json(); });
    }
    return productsPromise;
  }

  function read() {
    try {
      var raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return [];
      var parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed : [];
    } catch (e) {
      return [];
    }
  }

  function write(items) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
    // Storage events only fire on OTHER tabs; dispatch locally too.
    window.dispatchEvent(new CustomEvent('juicebar:cart-change'));
  }

  function count() {
    return read().reduce(function (n, it) { return n + (it.qty || 0); }, 0);
  }

  function add(handle, size, qty) {
    if (!handle) return;
    qty = Math.max(1, parseInt(qty, 10) || 1);
    var items = read();
    var existing = items.find(function (it) { return it.handle === handle && it.size === size; });
    if (existing) { existing.qty += qty; } else { items.push({ handle: handle, size: size || '', qty: qty }); }
    write(items);
  }

  function remove(handle, size) {
    var items = read().filter(function (it) { return !(it.handle === handle && it.size === (size || '')); });
    write(items);
  }

  function clear() { write([]); }

  function fmtCents(c) {
    var sign = c < 0 ? '-' : '';
    var n = Math.abs(c);
    return sign + '$' + Math.floor(n / 100) + '.' + String(n % 100).padStart(2, '0');
  }

  function resolve(items, catalog) {
    var byHandle = {};
    catalog.forEach(function (p) { byHandle[p.handle] = p; });
    return items.map(function (it) {
      var p = byHandle[it.handle];
      if (!p) return null;
      var unit = p.sale_price_cents > 0 ? p.sale_price_cents : p.price_cents;
      return {
        item: it,
        product: p,
        unitCents: unit,
        lineCents: unit * it.qty,
      };
    }).filter(Boolean);
  }

  function renderBadges() {
    var n = count();
    document.querySelectorAll('[data-cart-count]').forEach(function (el) {
      el.textContent = n > 0 ? ' (' + n + ')' : '';
    });
  }

  function renderCart(rootEl) {
    if (!rootEl) return;
    loadProducts().then(function (catalog) {
      var rows = resolve(read(), catalog);
      if (!rows.length) {
        rootEl.innerHTML = '<div class="empty-state"><p>Your cart is empty.</p><a href="/shop" class="btn btn--primary">Browse the shop</a></div>';
        return;
      }
      var subtotal = rows.reduce(function (n, r) { return n + r.lineCents; }, 0);
      var shipping = subtotal >= 5000 ? 0 : 799;
      var total = subtotal + shipping;
      var html = '<div class="table-wrap"><table class="table"><thead><tr><th>Product</th><th>Size</th><th>Qty</th><th class="cart-line-total">Line total</th><th></th></tr></thead><tbody>';
      rows.forEach(function (r) {
        html += '<tr>';
        html += '<td><a href="/products/' + r.product.handle + '">' + r.product.title + '</a></td>';
        html += '<td>' + (r.item.size || '&mdash;') + '</td>';
        html += '<td>' + r.item.qty + '</td>';
        html += '<td class="cart-line-total">' + fmtCents(r.lineCents) + '</td>';
        html += '<td><button type="button" class="remove-btn" data-handle="' + r.product.handle + '" data-size="' + (r.item.size || '') + '">Remove</button></td>';
        html += '</tr>';
      });
      html += '</tbody></table></div>';
      html += '<div class="cart-summary">';
      html += '<div class="cart-row"><span>Subtotal</span><span>' + fmtCents(subtotal) + '</span></div>';
      html += '<div class="cart-row text--muted"><span>Shipping</span><span>' + (shipping === 0 ? 'Free over $50' : fmtCents(shipping)) + '</span></div>';
      html += '<hr>';
      html += '<div class="cart-row cart-total"><span>Total</span><span>' + fmtCents(total) + '</span></div>';
      html += '<div class="cart-actions"><a href="/shop" class="btn">Keep shopping</a> <button type="button" class="btn btn--primary" id="checkout-btn">Checkout</button></div>';
      html += '</div>';
      rootEl.innerHTML = html;
      rootEl.querySelectorAll('.remove-btn').forEach(function (btn) {
        btn.addEventListener('click', function () {
          remove(btn.dataset.handle, btn.dataset.size);
        });
      });
      var checkout = document.getElementById('checkout-btn');
      if (checkout) checkout.addEventListener('click', function () { alert('This is a demo — checkout is not wired up.'); });
    });
  }

  function bindAddButtons() {
    document.querySelectorAll('[data-cart-add]').forEach(function (btn) {
      btn.addEventListener('click', function (ev) {
        ev.preventDefault();
        var handle = btn.dataset.cartAdd;
        var sizeSelect = document.getElementById('size-select');
        var qtySelect = document.getElementById('qty-select');
        var size = sizeSelect ? sizeSelect.value : (btn.dataset.size || '');
        var qty  = qtySelect ? qtySelect.value : 1;
        add(handle, size, qty);
        btn.textContent = 'Added';
        setTimeout(function () { btn.textContent = btn.dataset.label || 'Add to cart'; }, 1200);
      });
    });
  }

  function onReady() {
    bindAddButtons();
    renderBadges();
    renderCart(document.getElementById('cart-root'));
  }

  window.addEventListener('storage', function (e) { if (e.key === STORAGE_KEY) { renderBadges(); renderCart(document.getElementById('cart-root')); } });
  window.addEventListener('juicebar:cart-change', function () { renderBadges(); renderCart(document.getElementById('cart-root')); });

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', onReady);
  } else {
    onReady();
  }

  window.Cart = { add: add, remove: remove, clear: clear, count: count, read: read };
})();
