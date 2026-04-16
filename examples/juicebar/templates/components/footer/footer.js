// Newsletter signup stub: swap the submit label to confirm, no network call.
(function () {
  function init() {
    var form = document.querySelector('[data-newsletter-form]');
    if (!form) return;
    form.addEventListener('submit', function (event) {
      event.preventDefault();
      var button = form.querySelector('button');
      if (button) button.textContent = 'Thanks \u2014 see you next month';
    });
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
