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
        img.onload = null
      } else {
        checkTimeout = setTimeout(checkImage, 100)
      }
    }

    var sendImageSize = function() {
      clearTimeout(checkTimeout)
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

    img.onload = sendImageSize
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

render()
