(function () {
  function copyText(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text);
    }
    return new Promise(function (resolve, reject) {
      var ta = document.createElement('textarea');
      ta.value = text;
      ta.setAttribute('readonly', '');
      ta.style.position = 'absolute';
      ta.style.left = '-9999px';
      document.body.appendChild(ta);
      ta.select();
      try {
        document.execCommand('copy');
        resolve();
      } catch (err) {
        reject(err);
      } finally {
        document.body.removeChild(ta);
      }
    });
  }

  function wire(root) {
    var btn = root.querySelector('.copy-button__btn');
    var status = root.querySelector('.copy-button__status');
    if (!btn) return;
    btn.addEventListener('click', function () {
      var value = root.dataset.copyValue || '';
      copyText(value).then(function () {
        root.classList.add('copy-button--copied');
        if (status) status.textContent = 'Copied';
        setTimeout(function () {
          root.classList.remove('copy-button--copied');
          if (status) status.textContent = '';
        }, 1800);
      });
    });
  }

  document.querySelectorAll('.copy-button').forEach(wire);
})();
