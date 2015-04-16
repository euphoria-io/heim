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
    img.onload = function() {
      var ratio = img.width / img.height
      if (ratio < 9/16) {
        img.style.width = (9/16 * window.innerHeight) + 'px'
        img.style.height = 'auto'
      }
      window.top.postMessage({
        id: data.id,
        type: 'size',
        data: {
          width: img.width,
        }
      }, process.env.HEIM_ENDPOINT)
    }
    document.body.appendChild(img)
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
