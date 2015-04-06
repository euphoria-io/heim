var queryString = require('querystring')

function render() {
  var data = queryString.parse(location.search.substr(1))

  if (data.kind == 'youtube') {
    // jshint camelcase: false
    var embed = document.createElement('iframe')
    embed.src = '//www.youtube.com/embed/' + data.youtube_id + (data.autoplay ? '?autoplay=1' : '')
    document.body.appendChild(embed)
  }
}

render()
