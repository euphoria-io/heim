var queryString = require('querystring')

var allowedImageDomains = {
  'i.imgur.com': true,
  'imgs.xkcd.com': true,
  'i.ytimg.com': true,
}

function render() {
  var data = queryString.parse(location.search.substr(1))

  if (data.kind == 'img') {
    var domain = data.url.match(/\/\/([^/]+)\//)
    if (!domain || !allowedImageDomains.hasOwnProperty(domain[1])) {
      return
    }

    var img = document.createElement('img')
    img.src = data.url

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
        id: data.id,
        type: 'size',
        data: {
          width: displayWidth,
        }
      }, process.env.HEIM_ENDPOINT)
    }

    img.onload = function() {
      sendImageSize()

      // inspired by http://stackoverflow.com/a/4276742
      var frozenCanvas = document.createElement('canvas')
      var w = frozenCanvas.width = img.naturalWidth
      var h = frozenCanvas.height = img.naturalHeight
      frozenCanvas.getContext('2d').drawImage(img, 0, 0, w, h)
      document.body.appendChild(frozenCanvas)
      document.body.classList.add('frozen')
    }
    document.body.appendChild(img)
    checkImage()
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

window.addEventListener('message', function(ev) {
  if (ev.origin == process.env.HEIM_ENDPOINT) {
    if (ev.data.type == 'freeze') {
      document.body.classList.add('frozen')
    } else if (ev.data.type == 'unfreeze') {
      document.body.classList.remove('frozen')
    }
  }
}, false)

render()
