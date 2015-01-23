var _ = require('lodash')
var React = require('react')
var Autolinker = require('autolinker')


var autolinker = new Autolinker({
  twitter: false,
  truncate: 40,
  replaceFn: function(autolinker, match) {
    if (match.getType() == 'url') {
      var url = match.getUrl()
      var tag = autolinker.getTagBuilder().build(match)

      if (/^javascript/.test(url.toLowerCase())) {
        // Thanks, Jordan!
        return false
      }

      if (location.protocol == 'https:' && RegExp('^https?:\/\/' + location.hostname).test(url)) {
        // self-link securely
        tag.setAttr('href', url.replace(/^http:/, 'https:'))
      } else {
        tag.setAttr('rel', 'noreferrer')
      }

      return tag
    }
  },
})

module.exports = React.createClass({
  displayName: 'MessageText',

  mixins: [
    require('react-immutable-render-mixin'),
  ],

  render: function() {
    var html = _.escape(this.props.content)

    html = autolinker.link(html)

    return <span className={this.props.className} style={this.props.style} dangerouslySetInnerHTML={{
      __html: html
    }} />
  },
})
