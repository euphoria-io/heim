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
    document.body.style.backgroundImage = 'url(\'' + data.url + '\')'
    document.body.style.backgroundRepeat = 'no-repeat'
    document.body.style.backgroundSize = 'cover'
    document.body.style.backgroundPosition = 'left top'
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
