var queryString = require('querystring')


var allowedImageDomains = {
  'imgs.xkcd.com': true,
  'i.ytimg.com': true,
}

var frozen = true
var _freezeHandler = null
window.addEventListener('message', function(ev) {
  if (ev.origin == process.env.HEIM_ORIGIN) {
    if (ev.data.type == 'freeze') {
      frozen = true
    } else if (ev.data.type == 'unfreeze') {
      frozen = false
    }
    if (_freezeHandler) {
      _freezeHandler(frozen)
    }
  }
}, false)

function handleFreeze(callback) {
  _freezeHandler = callback
  callback(frozen)
}

function loadImage(id, url, onloadCallback) {
  var img = document.createElement('img')
  img.src = url

  var checkTimeout
  var checkImage = function() {
    if (img.naturalWidth) {
      sendImageSize()
    } else if (!widthSent) {
      checkTimeout = setTimeout(checkImage, 100)
    }
  }

  var widthSent = false
  var sendImageSize = function() {
    if (widthSent) {
      return
    }
    widthSent = true

    var displayHeight = window.innerHeight
    var displayWidth
    var ratio = img.naturalWidth / img.naturalHeight
    if (ratio < 9/16) {
      displayWidth = 9/16 * displayHeight
      img.style.width = displayWidth + 'px'
      img.style.height = 'auto'
    } else {
      displayWidth = img.naturalWidth * (displayHeight / img.naturalHeight)
    }

    window.top.postMessage({
      id: id,
      type: 'size',
      data: {
        width: displayWidth,
      }
    }, process.env.HEIM_ORIGIN)
  }

  img.onload = function() {
    sendImageSize()
    onloadCallback(img)
  }
  document.body.appendChild(img)
  checkImage()
}

function render() {
  var data = queryString.parse(location.search.substr(1))

  if (data.kind == 'imgur') {
    // jshint camelcase: false
    loadImage(data.id, '//i.imgur.com/' + data.imgur_id + 'l.jpg', function onload(img) {
      var fullSizeEl

      handleFreeze(function(frozen) {
        if (frozen) {
          if (fullSizeEl) {
            fullSizeEl.parentNode.removeChild(fullSizeEl)
          }
        } else {
          if (!fullSizeEl) {
            fullSizeEl = document.createElement('img')
            fullSizeEl.src = '//i.imgur.com/' + data.imgur_id + '.jpg'
            fullSizeEl.style.width = img.style.width
            fullSizeEl.style.height = img.style.height
            fullSizeEl.className = 'cover'
          }
          document.body.appendChild(fullSizeEl)
        }
      })
    })
  } else if (data.kind == 'img') {
    var domain = data.url.match(/\/\/([^/]+)\//)
    if (!domain || !allowedImageDomains.hasOwnProperty(domain[1])) {
      return
    }

    loadImage(data.id, data.url, function onload(img) {
      // inspired by http://stackoverflow.com/a/4276742
      var canvasEl = document.createElement('canvas')
      var w = canvasEl.width = img.naturalWidth
      var h = canvasEl.height = img.naturalHeight
      canvasEl.getContext('2d').drawImage(img, 0, 0, w, h)
      canvasEl.style.width = img.style.width
      canvasEl.style.height = img.style.height
      canvasEl.className = 'cover'
      document.body.appendChild(canvasEl)

      handleFreeze(function(frozen) {
        if (frozen) {
          document.body.appendChild(canvasEl)
        } else {
          canvasEl.parentNode.removeChild(canvasEl)
        }
      })
    })
  } else if (data.kind == 'youtube') {
    // jshint camelcase: false
    var embed = document.createElement('iframe')
    embed.src = '//www.youtube.com/embed/' + data.youtube_id + '?' + queryString.stringify({
      autoplay: data.autoplay,
      start: data.start,
    })
    document.body.appendChild(embed)
  }
}

render()
