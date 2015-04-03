var queryString = require('query-string')

function render() {
  if (location.origin != 'http://euphoria.local:8081' &&
      top.location.origin != 'https://euphoria.io' &&
      top.location.origin != 'http://euphoria.local:8080') {
    return
  }

  var data = queryString.parse(location.search)

  if (data.kind == 'youtube') {
    // jshint camelcase: false
    var embed = document.createElement('iframe')
    embed.src = 'http://www.youtube.com/embed/' + data.youtube_id + (data.autoplay ? '?autoplay=1' : '')
    document.body.appendChild(embed)
  }
}

render()
