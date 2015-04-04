var queryString = require('query-string')

function render() {
  var data = queryString.parse(location.search)

  if (data.kind == 'youtube') {
    // jshint camelcase: false
    var embed = document.createElement('iframe')
    embed.src = 'http://www.youtube.com/embed/' + data.youtube_id + (data.autoplay ? '?autoplay=1' : '')
    document.body.appendChild(embed)
  }
}

render()
