(function initFastTouch(body) {
  if ('ontouchstart' in window) {
    body.classList.add('touch')
    body.addEventListener('touchstart', function onFastTouchStart(ev) {
      ev.target.classList.add('touching')
    }, false)
    body.addEventListener('touchend', function onFastTouchEnd(ev) {
      ev.target.classList.remove('touching')
    }, false)
  }
}(document.body))
